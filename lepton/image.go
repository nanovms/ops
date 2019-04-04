package lepton

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-errors/errors"
)

// BuildImage builds a unikernel image for user
// supplied ELF binary.
func BuildImage(c Config) error {
	var err error
	m, err := BuildManifest(&c)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	if err = buildImage(&c, m); err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}

func createFile(filepath string) (*os.File, error) {
	path := path.Dir(filepath)
	var _, err = os.Stat(path)
	if os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}
	fd, err := os.Create(filepath)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	return fd, nil
}

// add /etc/resolv.conf
func addDNSConfig(m *Manifest, c *Config) {
	temp := getImageTempDir(c.Program)
	resolv := path.Join(temp, "resolv.conf")
	data := []byte("nameserver ")
	data = append(data, []byte(c.NameServer)...)
	err := ioutil.WriteFile(resolv, data, 0644)
	if err != nil {
		panic(err)
	}
	m.AddFile("/etc/resolv.conf", resolv)
}

// /proc/sys/kernel/hostname
func addHostName(m *Manifest, c *Config) {
	temp := getImageTempDir(c.Program)
	hostname := path.Join(temp, "hostname")
	data := []byte("uniboot")
	err := ioutil.WriteFile(hostname, data, 0644)
	if err != nil {
		panic(err)
	}
	m.AddFile("/proc/sys/kernel/hostname", hostname)
}

func addPasswd(m *Manifest, c *Config) {
	temp := getImageTempDir(c.Program)
	passwd := path.Join(temp, "passwd")
	data := []byte("root:x:0:0:root:/root:/bin/nobash")
	err := ioutil.WriteFile(passwd, data, 0644)
	if err != nil {
		panic(err)
	}
	m.AddFile("/etc/passwd", passwd)
}

// bunch of default files that's required.
func addDefaultFiles(m *Manifest, c *Config) error {

	commonPath := path.Join(GetOpsHome(), "common")
	if _, err := os.Stat(commonPath); os.IsNotExist(err) {
		os.MkdirAll(commonPath, 0755)
	} else if err != nil {
		return err
	}

	localtar := path.Join(GetOpsHome(), "common.tar.gz")
	if _, err := os.Stat(localtar); os.IsNotExist(err) {
		err := DownloadFile(localtar, commonArchive, 10)
		if err != nil {
			return err
		}
	}
	ExtractPackage(localtar, commonPath)

	localLibDNS := path.Join(commonPath, "libnss_dns.so.2")
	if _, err := os.Stat(localLibDNS); !os.IsNotExist(err) {
		m.AddFile(libDNS, localLibDNS)
	}

	localSslCert := path.Join(commonPath, "ca-certificates.crt")
	if _, err := os.Stat(localSslCert); !os.IsNotExist(err) {
		m.AddFile(sslCERT, localSslCert)
	}

	return nil
}

func addFilesFromPackage(packagepath string, m *Manifest) {

	rootPath := filepath.Join(packagepath, "sysroot")
	packageName := filepath.Base(packagepath)

	filepath.Walk(rootPath, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		filePath := strings.Split(hostpath, rootPath)
		m.AddFile(filePath[1], hostpath)
		return nil
	})

	filepath.Walk(packagepath, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == "sysroot" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		filePath := strings.Split(hostpath, packagepath)
		vmpath := filepath.Join(string(os.PathSeparator), packageName, filePath[1])
		m.AddFile(vmpath, hostpath)
		return nil
	})
}

func BuildPackageManifest(packagepath string, c *Config) (*Manifest, error) {
	m := NewManifest()

	m.program = c.Program
	addFromConfig(m, c)

	if len(c.Args) > 1 {
		if _, err := os.Stat(c.Args[1]); err == nil {
			m.AddFile(c.Args[1], c.Args[1])
		}
	}

	// Add files from package
	addFilesFromPackage(packagepath, m)
	return m, nil
}

func addFromConfig(m *Manifest, c *Config) {

	m.AddKernel(c.Kernel)
	addDNSConfig(m, c)
	addHostName(m, c)
	addPasswd(m, c)

	for _, f := range c.Files {
		m.AddFile(f, f)
	}
	for k, v := range c.MapDirs {
		addMappedFiles(k, v, m)
	}
	for _, d := range c.Dirs {
		m.AddDirectory(d)
	}
	for _, a := range c.Args {
		m.AddArgument(a)
	}

	for _, dbg := range c.Debugflags {
		m.AddDebugFlag(dbg, 't')
	}

	for k, v := range c.Env {
		m.AddEnvironmentVariable(k, v)
	}
}

func BuildManifest(c *Config) (*Manifest, error) {

	m := NewManifest()

	addFromConfig(m, c)
	m.AddUserProgram(c.Program)

	// run ldd and capture dependencies
	fmt.Println("Finding dependent shared libs")
	deps, err := getSharedLibs(c.TargetRoot, c.Program)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	for _, libpath := range deps {
		m.AddLibrary(libpath)
	}
	addDefaultFiles(m, c)
	return m, nil
}

func addMappedFiles(src string, dest string, m *Manifest) {
	dir, pattern := filepath.Split(src)
	filepath.Walk(dir, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		hostdir, filename := filepath.Split(hostpath)
		matched, _ := filepath.Match(pattern, filename)
		if matched {
			reldir, err := filepath.Rel(dir, hostdir)
			if err != nil {
				return err
			}
			vmpath := filepath.Join(dest, reldir, filename)
			m.AddFile(vmpath, hostpath)
		}
		return nil
	})
}

func buildImage(c *Config, m *Manifest) error {
	//  prepare manifest file
	var elfmanifest string

	elfmanifest = m.String()
	if c.ManifestName != "" {
		err := ioutil.WriteFile(c.ManifestName, []byte(elfmanifest), 0644)
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	// invoke mkfs to create the filesystem ie kernel + elf image
	createFile(mergedImg)

	defer cleanup(c)

	args := []string{}
	if c.TargetRoot != "" {
		args = append(args, "-r", c.TargetRoot)
	}
	args = append(args, mergedImg)
	mkfs := exec.Command(c.Mkfs, args...)
	stdin, err := mkfs.StdinPipe()
	if err != nil {
		return errors.Wrap(err, 1)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, elfmanifest)
	}()
	out, err := mkfs.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		return errors.Wrap(err, 1)
	}

	// produce final image, boot + kernel + elf
	fd, err := createFile(c.RunConfig.Imagename)
	defer func() {
		os.Remove(mergedImg)
		fd.Close()
	}()

	if err != nil {
		return errors.Wrap(err, 1)
	}
	catcmd := exec.Command("cat", c.Boot, mergedImg)
	catcmd.Stdout = fd
	err = catcmd.Start()
	if err != nil {
		return errors.Wrap(err, 1)
	}
	catcmd.Wait()
	return nil
}

func cleanup(c *Config) {
	temp := filepath.Base(c.Program) + "_temp"
	path := path.Join(GetOpsHome(), temp)
	os.RemoveAll(path)
}

type dummy struct {
	total uint64
}

func (bc dummy) Write(p []byte) (int, error) {
	return len(p), nil
}

func DownloadNightlyImages(c *Config) error {
	local, err := LocalTimeStamp()
	if err != nil {
		return err
	}
	remote, err := RemoteTimeStamp()
	if err != nil {
		return err
	}

	if _, err := os.Stat(NightlyLocalFolder); os.IsNotExist(err) {
		os.MkdirAll(NightlyLocalFolder, 0755)
	}
	localtar := path.Join(NightlyLocalFolder, nightlyFileName())
	// we have an update, let's download since it's nightly
	if remote != local || c.Force {
		if err = DownloadFile(localtar, NightlyReleaseUrl, 600); err != nil {
			return errors.Wrap(err, 1)
		}
		// update local timestamp
		updateLocalTimestamp(remote)
		ExtractPackage(localtar, NightlyLocalFolder)
	}

	// make mkfs executable
	err = os.Chmod(path.Join(NightlyLocalFolder, "mkfs"), 0775)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}

func DownloadReleaseImages(version string) error {
	url := getReleaseUrl(version)
	localFolder := getReleaseLocalFolder(version)
	if _, err := os.Stat(localFolder); os.IsNotExist(err) {
		os.MkdirAll(localFolder, 0755)
	}

	localtar := path.Join(localFolder, releaseFileName(version))

	if err := DownloadFile(localtar, url, 600); err != nil {
		return errors.Wrap(err, 1)
	}

	ExtractPackage(localtar, localFolder)

	// make mkfs executable
	err := os.Chmod(path.Join(localFolder, "mkfs"), 0775)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	updateLocalRelease(version)
	return nil
}

func DownloadFile(filepath string, url string, timeout int) error {
	fmt.Println("Downloading..", url)
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	c := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fsize, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

	// Create our progress reporter and pass it to be used alongside our writer
	counter := NewWriteCounter(fsize)
	counter.Start()

	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	counter.Finish()

	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}
	return nil
}

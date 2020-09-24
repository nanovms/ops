package lepton

import (
	"encoding/json"
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

var localImageDir = path.Join(GetOpsHome(), "images")

// BuildImage builds a unikernel image for user
// supplied ELF binary.
func BuildImage(c Config) error {
	m, err := BuildManifest(&c)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if err = buildImage(&c, m); err != nil {
		return errors.Wrap(err, 1)
	}

	err = saveImageConfig(c)
	if err != nil {
		return err
	}

	return nil
}

// rebuildImage rebuilds a unikernel image for user
// supplied ELF binary after volume attach/detach
func rebuildImage(c Config) error {
	c.Program = c.ProgramPath
	m, err := BuildManifest(&c)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if err = buildImage(&c, m); err != nil {
		return errors.Wrap(err, 1)
	}

	err = saveImageConfig(c)
	if err != nil {
		return err
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
	temp := getImageTempDir(c)
	resolv := path.Join(temp, "resolv.conf")
	data := []byte("nameserver ")
	data = append(data, []byte(c.NameServer)...)
	err := ioutil.WriteFile(resolv, data, 0644)
	if err != nil {
		panic(err)
	}
	err = m.AddFile("/etc/resolv.conf", resolv)
	if err != nil {
		panic(err)
	}
}

// /proc/sys/kernel/hostname
func addHostName(m *Manifest, c *Config) {
	temp := getImageTempDir(c)
	hostname := path.Join(temp, "hostname")
	data := []byte("uniboot")
	err := ioutil.WriteFile(hostname, data, 0644)
	if err != nil {
		panic(err)
	}
	err = m.AddFile("/proc/sys/kernel/hostname", hostname)
	if err != nil {
		panic(err)
	}
}

func addPasswd(m *Manifest, c *Config) {
	// Skip adding password file if present in package
	if m.FileExists("/etc/passwd") {
		return
	}
	temp := getImageTempDir(c)
	passwd := path.Join(temp, "passwd")
	data := []byte("root:x:0:0:root:/root:/bin/nobash")
	err := ioutil.WriteFile(passwd, data, 0644)
	if err != nil {
		panic(err)
	}
	err = m.AddFile("/etc/passwd", passwd)
	if err != nil {
		panic(err)
	}
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
		err := DownloadFileWithProgress(localtar, commonArchive, 10)
		if err != nil {
			return err
		}
	}
	ExtractPackage(localtar, commonPath)

	localLibDNS := path.Join(commonPath, "libnss_dns.so.2")
	if _, err := os.Stat(localLibDNS); !os.IsNotExist(err) {
		err = m.AddFile(libDNS, localLibDNS)
		if err != nil {
			return err
		}
	}

	localSslCert := path.Join(commonPath, "ca-certificates.crt")
	if _, err := os.Stat(localSslCert); !os.IsNotExist(err) {
		err = m.AddFile(sslCERT, localSslCert)
		if err != nil {
			return err
		}
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
		err = m.AddFile(filePath[1], hostpath)
		if err != nil {
			return err
		}
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
		err = m.AddFile(vmpath, hostpath)
		if err != nil {
			return err
		}
		return nil
	})
}

// BuildPackageManifest builds manifest using package
func BuildPackageManifest(packagepath string, c *Config) (*Manifest, error) {
	m := NewManifest(c.TargetRoot)

	// Add files from package
	addFilesFromPackage(packagepath, m)

	m.program = c.Program
	err := addFromConfig(m, c)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	if len(c.Args) > 1 {
		if _, err := os.Stat(c.Args[1]); err == nil {
			err = m.AddFile(c.Args[1], c.Args[1])
			if err != nil {
				return nil, err
			}
		}
	}

	return m, nil
}

func addFromConfig(m *Manifest, c *Config) error {
	m.AddKernel(c.Kernel)
	addDNSConfig(m, c)
	addHostName(m, c)
	addPasswd(m, c)

	for _, f := range c.Files {
		err := m.AddFile(f, f)
		if err != nil {
			return err
		}
	}

	for k, v := range c.MapDirs {
		err := addMappedFiles(k, v, m)
		if err != nil {
			return err
		}
	}

	for _, d := range c.Dirs {
		err := m.AddDirectory(d)
		if err != nil {
			return err
		}
	}

	for _, a := range c.Args {
		m.AddArgument(a)
	}

	if c.RebootOnExit {
		m.AddDebugFlag("reboot_on_exit", 't')
	}

	for _, dbg := range c.Debugflags {
		m.AddDebugFlag(dbg, 't')
	}

	for _, syscallName := range c.NoTrace {
		m.AddNoTrace(syscallName)
	}

	m.AddEnvironmentVariable("USER", "root")
	m.AddEnvironmentVariable("PWD", "/")
	for k, v := range c.Env {
		m.AddEnvironmentVariable(k, v)
	}

	for k, v := range c.Mounts {
		m.AddMount(k, v)
	}

	return nil
}

// BuildManifest builds manifest using config
func BuildManifest(c *Config) (*Manifest, error) {
	m := NewManifest(c.TargetRoot)

	err := addFromConfig(m, c)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	m.AddUserProgram(c.Program)

	deps, fileName, err := getSharedLibs(c.TargetRoot, c.Program)
	if err != nil {
		if len(fileName) == 0 {
			return nil, errors.New("library " + fileName + " not found")
		}
		return nil, errors.Wrap(err, 1)
	}
	for _, libpath := range deps {
		m.AddLibrary(libpath)
	}
	addDefaultFiles(m, c)
	return m, nil
}

func addMappedFiles(src string, dest string, m *Manifest) error {
	dir, pattern := filepath.Split(src)
	err := filepath.Walk(dir, func(hostpath string, info os.FileInfo, err error) error {
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
			err = m.AddFile(vmpath, hostpath)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
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

	// produce final image, boot + kernel + elf
	fd, err := createFile(c.RunConfig.Imagename)
	defer func() {
		fd.Close()
	}()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	defer cleanup(c)

	args := []string{}
	if c.TargetRoot != "" {
		args = append(args, "-r", c.TargetRoot)
	}

	if c.BaseVolumeSz != "" {
		args = append(args, "-s", c.BaseVolumeSz)
	}

	args = append(args, "-b", c.Boot)
	args = append(args, c.RunConfig.Imagename)

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
		log.Println("mkfs:" + string(out))
		return errors.Wrap(err, 1)
	}

	return nil
}

func cleanup(c *Config) {
	os.RemoveAll(c.BuildDir)
}

type dummy struct {
	total uint64
}

func (bc dummy) Write(p []byte) (int, error) {
	return len(p), nil
}

// DownloadNightlyImages downloads nightly build for nanos
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
		if err = DownloadFileWithProgress(localtar, NightlyReleaseURL, 600); err != nil {
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

// DownloadReleaseImages downloads nanos for particular release version
func DownloadReleaseImages(version string) error {
	url := getReleaseURL(version)
	localFolder := getReleaseLocalFolder(version)
	if _, err := os.Stat(localFolder); os.IsNotExist(err) {
		os.MkdirAll(localFolder, 0755)
	}

	localtar := path.Join(localFolder, releaseFileName(version))

	if err := DownloadFileWithProgress(localtar, url, 600); err != nil {
		return errors.Wrap(err, 1)
	}

	ExtractPackage(localtar, localFolder)

	// make mkfs executable
	err := os.Chmod(path.Join(localFolder, "mkfs"), 0775)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	updateLocalRelease(version)
	// FIXME hack to rename stage3.img to kernel.img
	oldKernel := path.Join(localFolder, "stage3.img")
	newKernel := path.Join(localFolder, "kernel.img")
	_, err = os.Stat(newKernel)
	if err == nil {
		return nil
	}
	_, err = os.Stat(oldKernel)
	if err == nil {
		os.Rename(oldKernel, newKernel)
	}

	return nil
}

// DownloadFileWithProgress downloads file using URL displaying progress counter
func DownloadFileWithProgress(filepath string, url string, timeout int) error {
	return DownloadFile(filepath, url, timeout, true)
}

// DownloadFile downloads file using URL
func DownloadFile(filepath string, url string, timeout int, showProgress bool) error {
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

	// Optionally create a progress reporter and pass it to be used alongside our writer
	if showProgress {
		counter := NewWriteCounter(fsize)
		counter.Start()
		_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
		counter.Finish()
	} else {
		_, err = io.Copy(out, resp.Body)
	}

	if err != nil {
		return err
	}

	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}
	return nil
}

func lookupFile(targetRoot string, path string) (string, error) {
	if targetRoot != "" {
		var targetPath string
		currentPath := path
		for {
			targetPath = filepath.Join(targetRoot, currentPath)
			fi, err := os.Lstat(targetPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return path, err
				}
				// lookup on host
				break
			}

			if fi.Mode()&os.ModeSymlink == 0 {
				// not a symlink found in target root
				return targetPath, nil
			}

			currentPath, err = os.Readlink(targetPath)
			if err != nil {
				return path, err
			}

			if currentPath[0] != '/' {
				// relative symlinks are ok
				path = targetPath
				break
			}

			// absolute symlinks need to be resolved again
		}
	}

	_, err := os.Stat(path)

	return path, err
}

// saveImageConfig saves image config as JSON
// for volume attach/detach purposes
func saveImageConfig(c Config) error {
	b, err := json.Marshal(c)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	imgFile := path.Base(c.RunConfig.Imagename)
	mnfName := strings.TrimSuffix(imgFile, "img") + "json"
	err = ioutil.WriteFile(path.Join(localManifestDir, mnfName), b, 0644)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}

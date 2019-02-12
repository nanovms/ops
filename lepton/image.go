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
	"runtime"
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
	filePath := path.Dir(filepath)
	var _, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			panic(err)
		}
	}
	fd, err := os.Create(filepath)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	return fd, nil
}

// add /etc/resolv.conf
func addDNSConfig(m *Manifest, c *Config) {
	data := []byte("nameserver ")
	data = append(data, []byte(c.NameServer)...)
	temp := path.Join(os.TempDir(), "resolv")
	err := ioutil.WriteFile(temp, data, 0666)
	if err != nil {
		panic(err)
	}
	//nameserver 127.0.1.1
	m.AddFile("/etc/resolv.conf", temp)
}

///proc/sys/kernel/hostname
func addHostName(m *Manifest, c *Config) {
	// uniboot is hardcoded in nanos virtio
	// may be better to handle 'proc/sys/kernel/hostname' open
	// in nanos as a spcial file like other device files
	data := []byte("uniboot")
	temp := path.Join(os.TempDir(), "hostname")
	err := ioutil.WriteFile(temp, data, 0666)
	if err != nil {
		panic(err)
	}
	//nameserver 127.0.1.1
	m.AddFile("/proc/sys/kernel/hostname", temp)
}

func addFilesFromPackage(packagepath string, m *Manifest) {

	rootpath := "sysroot"
	filepath.Walk(packagepath, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		parts := strings.Split(hostpath, string(os.PathSeparator))
		// files the need to go to root(/) are under sysroot in the package
		// rest of the files goes to /packagename folder
		if len(parts) > 2 && parts[2] == rootpath {
			m.AddFile(strings.Join(parts[3:], string(os.PathSeparator)), hostpath)
		} else {
			m.AddFile(strings.Join(parts[1:], string(os.PathSeparator)), hostpath)
		}
		return nil
	})
}

func BuildPackageManifest(packagepath string, c *Config) (*Manifest, error) {
	initDefaultImages(c)
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

	m.AddKernal(c.Kernel)
	addDNSConfig(m, c)
	addHostName(m, c)

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

	initDefaultImages(c)
	m := NewManifest()

	addFromConfig(m, c)
	m.AddUserProgram(c.Program)

	// run ldd and capture dependencies
	fmt.Println("Finding dependent shared libs")
	deps, err := getSharedLibs(c.Program)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	for _, libpath := range deps {
		m.AddLibrary(libpath)
	}
	return m, nil
}

func initDefaultImages(c *Config) {
	home, err := HomeDir()
	if err != nil {
		panic(err)
	}
	opsDirectory := path.Join(home, OpsDir)
	if c.Boot == "" {
		c.Boot = path.Join(opsDirectory, BootImg)
	}
	if c.Kernel == "" {
		c.Kernel = path.Join(opsDirectory, KernelImg)
	}
	if c.DiskImage == "" {
		c.DiskImage = FinalImg
	}
	if c.Mkfs == "" {
		c.Mkfs = path.Join(opsDirectory, Mkfs)
	}
	if c.NameServer == "" {
		// google dns server
		c.NameServer = "8.8.8.8"
	}
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
	// invoke mkfs to create the filesystem ie kernel + elf image
	createFile(mergedImg)
	mkfs := exec.Command(c.Mkfs, mergedImg)
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
	fd, err := createFile(c.DiskImage)
	defer fd.Close()
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

type dummy struct {
	total uint64
}

func (bc dummy) Write(p []byte) (int, error) {
	return len(p), nil
}

func DownloadBootImages(baseUrl string, force bool) error {
	home, err := HomeDir()
	if err != nil {
		return errors.Wrap(err, 1)
	}
	opsDirectory := path.Join(home, OpsDir)
	if _, err := os.Stat(opsDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(opsDirectory, 0755); err != nil {
			return errors.Wrap(err, 1)
		}
	}
	if baseUrl == "" {
		return DownloadImages(ReleaseBaseUrl, false, opsDirectory)
	}
	return DownloadImages(baseUrl, force, opsDirectory)
}

// DownloadImages downloads latest kernel images.
func DownloadImages(baseUrl string, force bool, baseDir string) error {

	mkfsFilePath := path.Join(baseDir, Mkfs)
	bootImgFilePath := path.Join(baseDir, BootImg)
	kernelImgFilePath := path.Join(baseDir, KernelImg)

	if _, err := os.Stat(mkfsFilePath); os.IsNotExist(err) || force {
		if err = DownloadFile(mkfsFilePath, fmt.Sprintf(baseUrl, path.Join(runtime.GOOS, "mkfs")), 600); err != nil {
			return errors.Wrap(err, 1)
		}
	}

	// make mkfs executable
	if err := os.Chmod(mkfsFilePath, 0775); err != nil {
		return errors.Wrap(err, 1)
	}

	if _, err := os.Stat(bootImgFilePath); os.IsNotExist(err) || force {
		if err = DownloadFile(bootImgFilePath, fmt.Sprintf(baseUrl, "boot.img"), 600); err != nil {
			return errors.Wrap(err, 1)
		}
	}

	if _, err := os.Stat(kernelImgFilePath); os.IsNotExist(err) || force {
		if err = DownloadFile(kernelImgFilePath, fmt.Sprintf(baseUrl, "stage3.img"), 600); err != nil {
			return errors.Wrap(err, 1)
		}
	}
	return nil
}

func DownloadFile(filepath string, url string, timeout int) error {

	fmt.Println("Downloading..", filepath)
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}()

	// Get the data
	c := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			panic(err)
		}
	}()

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

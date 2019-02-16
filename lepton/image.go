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
	rootPath := filepath.Join(packagepath, PackageRootPath)
	packageName := filepath.Base(packagepath)
	// Add files that should go to root(/) in image
	// These files are located inside package in directory const PackageRootPath
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
		if info.IsDir() && info.Name() == PackageRootPath {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}

		filePath := strings.Split(hostpath, packagepath)
		m.AddFile(filepath.Join(string(os.PathSeparator), packageName, filePath[1]), hostpath)
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
	deps, err := getSharedLibs(c.TargetRoot, c.Program)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	for _, libpath := range deps {
		m.AddLibrary(libpath)
	}
	return m, nil
}

func initDefaultImages(c *Config) {
	if c.Boot == "" {
		c.Boot = BootImg
	}
	if c.Kernel == "" {
		c.Kernel = KernelImg
	}
	if c.Mkfs == "" {
		c.Mkfs = Mkfs
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
	elfmanifest := m.String()

	// invoke mkfs to create the filesystem ie kernel + elf image
	mergedImgPath := filepath.Join(c.LocalTMPDir, mergedImg)
	if mergedImg, err := createFile(mergedImgPath); err != nil {
		return errors.Wrap(err, 1)
	} else {
		if err := mergedImg.Close(); err != nil {
			return errors.Wrap(err, 1)
		}
	}

	var args []string
	if c.TargetRoot != "" {
		args = append(args, "-r", c.TargetRoot)
	}
	args = append(args, mergedImgPath)

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

	bootImageOpen, err := os.Open(c.Boot)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer bootImageOpen.Close()

	mergedImgPathOpen, err := os.Open(mergedImgPath)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer mergedImgPathOpen.Close()

	// produce final image, boot + kernel + elf
	if c.DiskImage == "" {
		c.DiskImage = filepath.Join(c.LocalTMPDir, FinalImg)
	}
	fd, err := createFile(c.DiskImage)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer fd.Close()

	if _, err = io.Copy(fd, bootImageOpen); err != nil {
		return errors.Wrap(err, 1)
	}
	if _, err = io.Copy(fd, mergedImgPathOpen); err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}

type dummy struct {
	total uint64
}

func (bc dummy) Write(p []byte) (int, error) {
	return len(p), nil
}

func DownloadBootImages() error {
	return DownloadImages(ReleaseBaseUrl, false)
}

// DownloadImages downloads latest kernel images.
func DownloadImages(baseUrl string, force bool) error {
	var err error
	if _, err := os.Stat(".staging"); os.IsNotExist(err) {
		os.MkdirAll(".staging", 0755)
	}

	if _, err = os.Stat(Mkfs); os.IsNotExist(err) || force {
		if err = DownloadFile(Mkfs, fmt.Sprintf(baseUrl, path.Join(runtime.GOOS, "mkfs")), 600); err != nil {
			return errors.Wrap(err, 1)
		}
	}

	// make mkfs executable
	err = os.Chmod(Mkfs, 0775)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if _, err = os.Stat(BootImg); os.IsNotExist(err) || force {
		if err = DownloadFile(BootImg, fmt.Sprintf(baseUrl, "boot.img"), 600); err != nil {
			return errors.Wrap(err, 1)
		}
	}

	if _, err = os.Stat(KernelImg); os.IsNotExist(err) || force {
		if err = DownloadFile(KernelImg, fmt.Sprintf(baseUrl, "stage3.img"), 600); err != nil {
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

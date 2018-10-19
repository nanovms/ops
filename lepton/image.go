package lepton

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/go-errors/errors"
)

// BuildImage builds a unikernel image for user
// supplied ELF binary.
func BuildImage(userImage string, c Config) error {
	var err error
	if err = buildImage(userImage, &c); err != nil {
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

func buildManifest(userImage string, c *Config) (*Manifest, error) {

	m := NewManifest()
	m.AddUserProgram(userImage)
	m.AddKernal(c.Kernel)

	// run ldd and capture dependencies
	deps, err := getSharedLibs(userImage)
	fmt.Println(deps)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	for _, libpath := range deps {
		m.AddLibrary(libpath)
	}
	//
	return m, nil
}

func initDefaultImages(c *Config) {
	if c.Boot == "" {
		c.Boot = bootImg
	}
	if c.Kernel == "" {
		c.Kernel = kernelImg
	}
	if c.DiskImage == "" {
		c.DiskImage = FinalImg
	}
	if c.Mkfs == "" {
		c.Mkfs = Mkfs
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
		_, filename := filepath.Split(hostpath)
		matched, _ := filepath.Match(pattern, filename)
		if matched {
			vmpath := filepath.Join(dest, filename)
			m.AddFile(vmpath, hostpath)
		}
		return nil
	})
}

func buildImage(userImage string, c *Config) error {
	//  prepare manifest file
	var elfmanifest string
	initDefaultImages(c)
	m, err := buildManifest(userImage, c)
	if err != nil {
		return errors.Wrap(err, 1)
	}
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
	elfmanifest = m.String()
	fmt.Println(elfmanifest)

	// invoke mkfs to create the filesystem ie kernel + elf image
	createFile(mergedImg)
	mkfs := exec.Command(c.Mkfs, mergedImg)
	stdin, err := mkfs.StdinPipe()
	if err != nil {
		return errors.Wrap(err, 1)
	}
	_, err = io.WriteString(stdin, elfmanifest)
	if err != nil {
		return errors.Wrap(err, 1)
	}
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

func DownloadBootImages() error {
	return DownloadImages(dummy{}, ReleaseBaseUrl)
}

// DownloadImages downloads latest kernel images.
func DownloadImages(w io.Writer, baseUrl string) error {
	var err error
	if _, err := os.Stat(".staging"); os.IsNotExist(err) {
		os.MkdirAll(".staging", 0755)
	}

	if _, err = os.Stat(".staging/mkfs"); os.IsNotExist(err) {
		if err = downloadFile(".staging/mkfs", fmt.Sprintf(baseUrl, "mkfs"), w); err != nil {
			return errors.Wrap(err, 1)
		}
	}

	// make mkfs executable
	err = os.Chmod(".staging/mkfs", 0775)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if _, err = os.Stat(".staging/boot"); os.IsNotExist(err) {
		if err = downloadFile(".staging/boot", fmt.Sprintf(baseUrl, "boot"), w); err != nil {
			return errors.Wrap(err, 1)
		}
	}

	if _, err = os.Stat(".staging/stage3"); os.IsNotExist(err) {
		if err = downloadFile(".staging/stage3", fmt.Sprintf(baseUrl, "stage3"), w); err != nil {
			return errors.Wrap(err, 1)
		}
	}
	return nil
}

func downloadFile(filepath string, url string, w io.Writer) error {
	// download to a temp file and later rename it
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer resp.Body.Close()
	// progress reporter.
	_, err = io.Copy(out, io.TeeReader(resp.Body, w))
	if err != nil {
		return errors.Wrap(err, 1)
	}
	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}

package lepton

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/fs"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

// LocalImageDir is the directory where ops save images
var LocalImageDir = path.Join(GetOpsHome(), "images")

// BuildImage builds a unikernel image for user
// supplied ELF binary.
func BuildImage(c types.Config) error {

	m, err := BuildManifest(&c)
	if err != nil {
		return fmt.Errorf("failed building manifest: %v", err)
	}

	if err = createImageFile(&c, m); err != nil {
		return fmt.Errorf("failed creating image file: %v", err)
	}

	return nil
}

// rebuildImage rebuilds a unikernel image for user
// supplied ELF binary after volume attach/detach
func rebuildImage(c types.Config) error {
	c.Program = c.ProgramPath
	m, err := BuildManifest(&c)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if err = createImageFile(&c, m); err != nil {
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
func addDNSConfig(m *fs.Manifest, c *types.Config) {
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
func addHostName(m *fs.Manifest, c *types.Config) {
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

func addPasswd(m *fs.Manifest, c *types.Config) {
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
func addCommonFilesToManifest(m *fs.Manifest) error {

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
	ExtractPackage(localtar, commonPath, NewConfig())

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

func addFilesFromPackage(packagepath string, m *fs.Manifest) {

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
func BuildPackageManifest(packagepath string, c *types.Config) (*fs.Manifest, error) {
	m := fs.NewManifest(c.TargetRoot)

	addFilesFromPackage(packagepath, m)

	m.SetProgram(c.Program)

	err := setManifestFromConfig(m, c)
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

func setManifestFromConfig(m *fs.Manifest, c *types.Config) error {
	m.AddKernel(c.Kernel)
	addDNSConfig(m, c)
	addHostName(m, c)
	addPasswd(m, c)
	m.SetKlibDir(getKlibsDir(c.NightlyBuild))
	m.AddKlibs(c.RunConfig.Klibs)

	for _, f := range c.Files {
		hostPath := path.Join(c.LocalFilesParentDirectory, f)
		filePath := f
		err := m.AddFile(filePath, hostPath)
		if err != nil {
			return err
		}
	}

	for k, v := range c.MapDirs {
		err := addMappedFiles(k, v, c.LocalFilesParentDirectory, m)
		if err != nil {
			return err
		}
	}

	for _, d := range c.Dirs {
		err := m.AddDirectory(d, c.LocalFilesParentDirectory)
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
	m.AddEnvironmentVariable("OPS_VERSION", Version)
	m.AddEnvironmentVariable("NANOS_VERSION", c.NanosVersion)
	for k, v := range c.Env {
		m.AddEnvironmentVariable(k, v)
	}

	if _, hasRadarKey := c.Env["RADAR_KEY"]; hasRadarKey {
		m.AddKlibs([]string{"tls", "radar"})

		if _, hasRadarImageName := c.Env["RADAR_IMAGE_NAME"]; !hasRadarImageName {
			m.AddEnvironmentVariable("RADAR_IMAGE_NAME", c.CloudConfig.ImageName)
		}
	}

	for k, v := range c.Mounts {
		m.AddMount(k, v)
	}

	if c.RunConfig.IPAddress != "" || c.RunConfig.IPv6Address != "" {
		m.AddNetworkConfig(&fs.ManifestNetworkConfig{
			IP:      c.RunConfig.IPAddress,
			IPv6:    c.RunConfig.IPv6Address,
			Gateway: c.RunConfig.Gateway,
			NetMask: c.RunConfig.NetMask,
		})
	}

	if len(c.RunConfig.Ports) != 0 {
		m.AddEnvironmentVariable("OPS_PORT", strings.Join(c.RunConfig.Ports, ","))
	}

	return nil
}

// BuildManifest builds manifest using config
func BuildManifest(c *types.Config) (*fs.Manifest, error) {
	m := fs.NewManifest(c.TargetRoot)

	addCommonFilesToManifest(m)

	err := m.AddUserProgram(c.Program)
	if err != nil {
		return nil, err
	}

	err = setManifestFromConfig(m, c)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	deps, err := getSharedLibs(c.TargetRoot, c.Program)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	for _, libpath := range deps {
		m.AddLibrary(libpath)
	}

	if c.RunConfig.IPAddress != "" || c.RunConfig.IPv6Address != "" {
		m.AddNetworkConfig(&fs.ManifestNetworkConfig{
			IP:      c.RunConfig.IPAddress,
			IPv6:    c.RunConfig.IPv6Address,
			Gateway: c.RunConfig.Gateway,
			NetMask: c.RunConfig.NetMask,
		})
	}

	for k, v := range c.ManifestPassthrough {
		m.AddPassthrough(k, v)
	}

	return m, nil
}

func addMappedFiles(src string, dest string, workDir string, m *fs.Manifest) error {
	dir, pattern := filepath.Split(src)
	parentDir := filepath.Base(dir)
	err := filepath.Walk(dir, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		hostdir, filename := filepath.Split(hostpath)
		matched, _ := filepath.Match(pattern, filename)
		if matched {
			if info.IsDir() {
				addedDir := parentDir
				hostBase := filepath.Base(hostpath)
				if hostBase != parentDir {
					filepath.Join(parentDir, hostBase)
				}
				return m.AddDirectory(addedDir, workDir)
			}

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

func createImageFile(c *types.Config, m *fs.Manifest) error {
	// produce final image, boot + kernel + elf
	fd, err := createFile(c.RunConfig.Imagename)
	defer func() {
		fd.Close()
	}()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	defer cleanup(c)

	mkfsCommand := fs.NewMkfsCommand(m)

	if c.BaseVolumeSz != "" {
		mkfsCommand.SetFileSystemSize(c.BaseVolumeSz)
	}

	mkfsCommand.SetBoot(c.Boot)
	if c.Uefi {
		if c.UefiBoot == "" {
			return errors.New("this Nanos version does not support UEFI, consider changing image type")
		}
		mkfsCommand.SetUefi(c.UefiBoot)
	}
	mkfsCommand.SetFileSystemPath(c.RunConfig.Imagename)

	err = mkfsCommand.Execute()
	if err != nil {
		return err
	}

	return nil
}

func cleanup(c *types.Config) {
	os.RemoveAll(c.BuildDir)
}

type dummy struct {
	total uint64
}

func (bc dummy) Write(p []byte) (int, error) {
	return len(p), nil
}

// DownloadNightlyImages downloads nightly build for nanos
func DownloadNightlyImages(c *types.Config) error {
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
		ExtractPackage(localtar, NightlyLocalFolder, c)
	}

	return nil
}

// DownloadCommonFiles dowloads common tarball files and extract them to common directory
func DownloadCommonFiles() error {
	commonPath := path.Join(GetOpsHome(), "common")
	if _, err := os.Stat(commonPath); os.IsNotExist(err) {
		os.MkdirAll(commonPath, 0755)
	} else if err != nil {
		return err
	}

	localtar := path.Join(GetOpsHome(), "common.tar.gz")
	err := DownloadFileWithProgress(localtar, commonArchive, 10)
	if err != nil {
		return err
	}
	ExtractPackage(localtar, commonPath, NewConfig())
	return nil
}

// CheckNanosVersionExists verifies whether version exists in filesystem
func CheckNanosVersionExists(version string) (bool, error) {
	_, err := os.Stat(path.Join(GetOpsHome(), version))
	if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// DownloadReleaseImages downloads nanos for particular release version
func DownloadReleaseImages(version string) error {
	url := getReleaseURL(version)
	localFolder := getReleaseLocalFolder(version)

	localtar := path.Join("/tmp", releaseFileName(version))
	defer os.Remove(localtar)

	if err := DownloadFileWithProgress(localtar, url, 600); err != nil {
		return errors.Wrap(err, 1)
	}

	if _, err := os.Stat(localFolder); os.IsNotExist(err) {
		os.MkdirAll(localFolder, 0755)
	}

	ExtractPackage(localtar, localFolder, NewConfig())

	// FIXME hack to rename stage3.img to kernel.img
	oldKernel := path.Join(localFolder, "stage3.img")
	newKernel := path.Join(localFolder, "kernel.img")
	_, err := os.Stat(newKernel)
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
	log.Info("Downloading..", url)
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}

	// Get the data
	c := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		os.Remove(out.Name())
		return errors.New("resource not found")
	}

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

	out.Close()
	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}

	return nil
}

// CreateArchive compress files into an archive
func CreateArchive(archive string, files []string) error {
	fd, err := os.Create(archive)
	if err != nil {
		return err
	}

	gzw := gzip.NewWriter(fd)

	tw := tar.NewWriter(gzw)

	for _, file := range files {
		fstat, err := os.Stat(file)
		if err != nil {
			return err
		}

		// write the header
		if err := tw.WriteHeader(&tar.Header{
			Name:   filepath.Base(file),
			Mode:   int64(fstat.Mode()),
			Size:   fstat.Size(),
			Format: tar.FormatGNU,
		}); err != nil {
			return err
		}

		fi, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data to tar
		if _, err := io.CopyN(tw, fi, fstat.Size()); err != nil {
			return err
		}
		if err = fi.Close(); err != nil {
			return err
		}
	}

	// Explicitly close all writers in correct order without any error
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gzw.Close(); err != nil {
		return err
	}
	if err := fd.Close(); err != nil {
		return err
	}
	return nil
}

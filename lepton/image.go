package lepton

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
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

// BuildImageFromPackage builds nanos image using a package
func BuildImageFromPackage(packagepath string, c types.Config) error {
	m, err := BuildPackageManifest(packagepath, &c)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if err := createImageFile(&c, m); err != nil {
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

	var data []byte
	for _, ns := range c.NameServers {
		data = append(data, []byte(fmt.Sprintln("nameserver ", ns))...)
	}
	err := os.WriteFile(resolv, data, 0644)
	if err != nil {
		fmt.Printf("Failed save dns in temporary resolv.conf, error: %s", err)
		os.Exit(1)
	}
	err = m.AddFile("/etc/resolv.conf", resolv)
	if err != nil {
		fmt.Printf("Failed add resolv.conf, error: %s", err)
		os.Exit(1)
	}
}

// /proc/sys/kernel/hostname
func addHostName(m *fs.Manifest, c *types.Config) {
	temp := getImageTempDir(c)
	hostname := path.Join(temp, "hostname")
	data := []byte("uniboot")
	err := os.WriteFile(hostname, data, 0644)
	if err != nil {
		fmt.Printf("Failed save hostname tmp file, error: %s", err)
		os.Exit(1)
	}
	err = m.AddFile("/proc/sys/kernel/hostname", hostname)
	if err != nil {
		fmt.Printf("Failed add hostname, error: %s", err)
		os.Exit(1)
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
	err := os.WriteFile(passwd, data, 0644)
	if err != nil {
		fmt.Printf("Failed save passwd in temporary file, error: %s", err)
		os.Exit(1)
	}
	err = m.AddFile("/etc/passwd", passwd)
	if err != nil {
		fmt.Printf("Failed add passwd, error: %s", err)
		os.Exit(1)
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

		filePath := strings.Split(hostpath, rootPath)
		if info.IsDir() {
			m.MkdirPath(strings.TrimPrefix(filePath[1], string(filepath.Separator)))
			return nil
		}

		err = m.AddFile(filePath[1], hostpath)
		if err != nil {
			log.Error(err)
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
			log.Error(err)
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

	if !c.DisableArgsCopy && len(c.Args) > 1 {
		if f, err := os.Stat(c.Args[1]); err == nil {
			if f.IsDir() {
				err = m.AddDirectory(c.Args[1], c.Args[1])
			} else {
				err = m.AddFile(c.Args[1], c.Args[1])
			}

			if err != nil {
				return nil, err
			}
		}
	}

	for k, v := range c.ManifestPassthrough {
		m.AddPassthrough(k, v)
	}

	return m, nil
}

func setManifestFromConfig(m *fs.Manifest, c *types.Config) error {
	m.AddKernel(c.Kernel)
	addDNSConfig(m, c)
	addHostName(m, c)
	addPasswd(m, c)
	if c.KlibDir != "" {
		m.SetKlibDir(c.KlibDir)
	} else {
		m.SetKlibDir(getKlibsDir(c.NightlyBuild, c.NanosVersion))
	}

	m.AddKlibs(c.Klibs)

	for _, f := range c.Files {
		hostPath := f

		if filepath.IsAbs(f) {
			hostPath = filepath.Join(c.TargetRoot, f)
		} else {
			hostPath = path.Join(c.LocalFilesParentDirectory, f)
		}

		err := m.AddFile(f, hostPath)
		if err != nil {
			return err
		}
	}

	for k, v := range c.MapDirs {
		for _, x := range c.Args {
			if x == filepath.Base(v) {
				errstr := fmt.Sprintf("can't have directory with same name as binary %v", x)
				return errors.New(errstr)
			}
		}
		if filepath.IsAbs(k) {
			k = filepath.Join(c.TargetRoot, k)
		} else {
			k = filepath.Join(c.LocalFilesParentDirectory, k)
		}
		err := addMappedFiles(k, v, m)
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
	m.AddEnvironmentVariable("IMAGE_NAME", c.CloudConfig.ImageName)
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

	deps, err := getSharedLibs(c.TargetRoot, c.Program, c)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	for libpath, hostpath := range deps {
		m.AddFile(libpath, hostpath)
	}

	// legacy sigle nic/instance method
	if c.RunConfig.IPAddress != "" || c.RunConfig.IPv6Address != "" {
		m.AddNetworkConfig(&fs.ManifestNetworkConfig{
			IP:      c.RunConfig.IPAddress,
			IPv6:    c.RunConfig.IPv6Address,
			Gateway: c.RunConfig.Gateway,
			NetMask: c.RunConfig.NetMask,
		})
	}

	// new many nics/instance
	// only for proxmox atm
	// this overrides anything in legacy RunConfig ip address setting
	nics := c.RunConfig.Nics
	for i := 0; i < len(nics); i++ {
		if i == 0 {
			m.AddNetworkConfig(&fs.ManifestNetworkConfig{
				IP:      nics[i].IPAddress,
				Gateway: nics[i].Gateway,
				NetMask: nics[i].NetMask,
			})
		} else {
			ifaces := make(map[string]interface{})

			// only set if ip given otherwise assume dhcp
			ifaces["ipaddr"] = nics[i].IPAddress
			ifaces["netmask"] = nics[i].NetMask
			ifaces["gateway"] = nics[i].Gateway

			s := strconv.Itoa(i + 1)
			m.AddPassthrough("en"+s, ifaces)
		}
	}

	for k, v := range c.ManifestPassthrough {
		m.AddPassthrough(k, v)
	}

	return m, nil
}

func addMappedFiles(src string, dest string, m *fs.Manifest) error {
	dir, pattern := filepath.Split(src)
	err := filepath.Walk(dir, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		hostdir, filename := filepath.Split(hostpath)
		matched, _ := filepath.Match(pattern, filename)
		if matched {
			reldir, err := filepath.Rel(dir, hostdir)
			if err != nil {
				return err
			}
			vmpath := filepath.Join(dest, reldir, filename)

			if info.IsDir() {
				m.MkdirPath(vmpath)
				return nil
			}

			if (info.Mode() & os.ModeSymlink) == os.ModeSymlink {
				return m.AddLink(vmpath, hostpath)
			}
			return m.AddFile(vmpath, hostpath)
		}
		return nil
	})
	return err
}

func createImageFile(c *types.Config, m *fs.Manifest) error {
	// produce final image, boot + kernel + elf
	fd, err := createFile(c.RunConfig.ImageName)
	defer func() {
		fd.Close()
	}()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	defer cleanup(c)

	if c.RunConfig.ShowDebug {
		fmt.Printf("Manifest:\n\t%+v\n", m)
	}

	mkfsCommand := fs.NewMkfsCommand(m, true)

	if c.BaseVolumeSz != "" {
		mkfsCommand.SetFileSystemSize(c.BaseVolumeSz)
	}

	mkfsCommand.SetBoot(c.Boot)
	if c.Uefi {
		if c.UefiBoot == "" {
			return errors.New("this Nanos version does not support UEFI, consider changing image type")
		}

		if strings.Contains(c.Kernel, "arm") {
			c.UefiBoot = strings.Replace(c.UefiBoot, "/bootx64.efi", "-arm/bootaa64.efi", -1)
		}

		mkfsCommand.SetUefi(c.UefiBoot)
	}

	mkfsCommand.SetFileSystemPath(c.RunConfig.ImageName)

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
// arch defaults to x86-64 if empty
func DownloadReleaseImages(version string, arch string) error {
	url := getReleaseURL(version)
	if arch == "arm" {
		url = strings.Replace(url, ".tar.gz", "-virt.tar.gz", -1)
	}

	localFolder := getReleaseLocalFolder(version)

	if arch == "arm" {
		localFolder = localFolder + "-arm"
	}

	localtar := path.Join("/tmp", releaseFileName(version))
	defer os.Remove(localtar)

	if err := DownloadFileWithProgress(localtar, url, 600); err != nil {

		if strings.Index(err.Error(), "can not download file") > -1 {
			return fmt.Errorf("release '%s' is not found", version)
		}

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
func DownloadFile(fpath string, url string, timeout int, showProgress bool) error {
	out, err := os.CreateTemp(filepath.Dir(fpath), fmt.Sprintf("*%s", filepath.Base(fpath)))
	if err != nil {
		return err
	}

	// we dont care about the error here
	creds, _ := ReadCredsFromLocal()

	// Get the data
	c := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	req, err := BaseHTTPRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if creds != nil {
		req.Header.Set(APIKeyHeader, creds.APIKey)
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		os.Remove(out.Name())
		return fmt.Errorf("can not download file from: %s, status: %s", url, resp.Status)
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
	err = os.Rename(out.Name(), fpath)
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

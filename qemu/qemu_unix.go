//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package qemu

import (
	"bufio"
	"bytes"
	"crypto/rand"
	mrand "math/rand"
	"time"

	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

// GenMgmtPort should generate mgmt before launching instance and
// persist in instance metadata
func GenMgmtPort() string {
	dd := mrand.Int31n(10000) + 40000
	return strconv.Itoa(int(dd))
}

func qemuBaseCommand() string {
	x86 := "qemu-system-x86_64"
	arm := "qemu-system-aarch64"

	if MACPKGD != "" {
		arm = "/usr/local/bin/qemu-system-aarch64"
		x86 = "/usr/local/bin/qemu-system-x86_64"
	}

	if lepton.AltGOARCH != "" {
		if lepton.AltGOARCH == "amd64" {
			return x86
		}
		if lepton.AltGOARCH == "arm64" {
			return arm
		}
	}

	if lepton.RealGOARCH == "amd64" {
		return x86
	}

	if lepton.RealGOARCH == "arm64" {
		return arm
	}

	return x86
}

type qemu struct {
	cmd        *exec.Cmd
	drives     []drive
	devices    []device
	ifaces     []netdev
	display    display
	serial     serial
	flags      []string
	kernel     string
	atExitHook string
	mgmt       string // not sure why we have this as string..
}

func newQemu() Hypervisor {
	return &qemu{}
}

func (q *qemu) SetKernel(kernel string) {
	q.kernel = kernel
}

func (q *qemu) Stop() {
	if q.cmd != nil {

		commands := []string{
			`{ "execute": "qmp_capabilities" }`,
			`{ "execute": "system_powerdown" }`,
		}

		if q.mgmt != "" {
			ExecuteQMP(commands, q.mgmt)
			time.Sleep(2 * time.Second)
		}

		if err := q.cmd.Process.Kill(); err != nil {
			log.Error(err)
		}

		// do not print errors as the command could be started with Run()
		q.cmd.Wait()
	}
}

func logv(rconfig *types.RunConfig, msg string) {
	if rconfig.Verbose {
		log.Info(msg)
	}
}

func (q *qemu) Command(rconfig *types.RunConfig) *exec.Cmd {
	args, err := q.Args(rconfig)
	if err != nil {
		log.Error(err)
		return nil
	}
	if q.atExitHook != "" {
		qc := qemuBaseCommand() + " " + strings.Join(args, " ")
		fullCmd := qc + "; " + q.atExitHook
		logv(rconfig, fullCmd)
		q.cmd = exec.Command("/bin/sh", "-c", fullCmd)
	} else {
		logv(rconfig, qemuBaseCommand()+" "+strings.Join(args, " "))

		q.cmd = exec.Command(qemuBaseCommand(), args...)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func(chan os.Signal) {
		<-c
		q.Stop()
	}(c)

	return q.cmd
}

func (q *qemu) Start(rconfig *types.RunConfig) error {
	if q.cmd == nil {
		q.Command(rconfig)
		if q.cmd == nil {
			return errors.New("Failed to create Qemu command line")
		}
		q.cmd.Stdout = os.Stdout
		q.cmd.Stderr = os.Stderr
	}

	if rconfig.QMP {
		q.mgmt = rconfig.Mgmt
	}

	if rconfig.Background {
		err := q.cmd.Start()
		if err != nil {
			log.Error(err)
		}
	} else {
		q.cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
		if err := q.cmd.Run(); err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (q *qemu) addDrive(id, image, ifaceType string) {
	drv := drive{
		path:   image,
		iftype: ifaceType,
		ID:     id,
	}
	fileExt := filepath.Ext(image)
	if !strings.Contains(fileExt, "qcow") {
		drv.format = "raw"
	}
	q.drives = append(q.drives, drv)
}

func (q *qemu) addDisplay(dispType string) {
	q.display = display{disptype: dispType}
}

func (q *qemu) addVfioPci(pciClass string, count int) error {
	pciDevs, err := exec.Command("lspci", "-nDk").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to list PCI devices: %v", err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(pciDevs))
	detected := 0
	var detectedAddress string
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "\t") {
			var found bool
			devAddress, remainder, found := strings.Cut(line, " ")
			if !found {
				return fmt.Errorf("Failed to find PCI device address in line '%s'", line)
			}
			if strings.HasPrefix(remainder, pciClass) {
				detectedAddress = devAddress
			}
		} else if detectedAddress != "" && strings.Contains(line, "driver") && strings.Contains(line, "vfio-pci") {
			q.devices = append(q.devices, device{
				driver:  "vfio-pci,rombar=0",
				devtype: "host",
				devid:   detectedAddress,
			})
			detected++
			if detected == count {
				break
			}
			detectedAddress = ""
		}
	}
	if err = scanner.Err(); err != nil {
		return fmt.Errorf("Failed to scan PCI device list: %v", err)
	}
	if detected < count {
		return fmt.Errorf(`Failed to find %d PCI device(s) with class %s (found %d)
Ensure the vfio-pci Linux kernel driver is loaded and bound to at least %d device(s)`, count, pciClass, detected, count)
	}
	return nil
}

func (q *qemu) addSerial(serialType string) {
	q.serial = serial{serialtype: serialType}
}

// OPSD is injected at release build time to enable vmnet-bridged
// this is only for a packaged macos release that wish to run N
// instances locally.
var OPSD = ""

// MACPKGD is a flag for apps that have been packed for a macos installer.
// It is injected at release build time to enable vmnet-bridged and location of fw.
var MACPKGD = ""

// addDevice adds a device to the qemu for rendering to string arguments. If the
// devType is "user" then the ifaceName is ignored and host forward ports are
// added. If the mac address is empty then a random mac address is chosen.
// Backend interface are created for each device and their ids are auto
// incremented.
func (q *qemu) addNetDevice(devType, ifaceName, mac string, hostPorts []string, udpPorts []string) {
	if OPSD != "" {
		dv := device{
			driver:  "virtio-net-pci",
			devtype: "netdev=vmnet",
		}

		ndv := netdev{
			nettype: "vmnet-bridged",
			id:      "vmnet",
		}

		dv.mac = generateMac()
		ndv.ifname = "en0"

		q.devices = append(q.devices, dv)
		q.ifaces = append(q.ifaces, ndv)

	} else {
		i := len(q.ifaces)
		si := strconv.Itoa(i)

		id := fmt.Sprintf("n%d", i)
		dv := device{
			driver:  "virtio-net",
			devtype: "netdev",
			devid:   id,
			addr:    "0x" + si,
		}
		ndv := netdev{
			nettype: devType,
			id:      id,
		}

		if mac == "" {
			dv.mac = generateMac()
		}

		if devType != "user" {
			ndv.ifname = ifaceName
		} else {
			// hack - FIXME
			// this is user-mode
			if i == 0 {
				for _, p := range hostPorts {
					ndv.hports = append(ndv.hports, portfwd{port: p, proto: "tcp"})
				}
				for _, p := range udpPorts {
					ndv.hports = append(ndv.hports, portfwd{port: p, proto: "udp"})
				}
			}
		}

		q.devices = append(q.devices, dv)
		q.ifaces = append(q.ifaces, ndv)
	}
}

func (q *qemu) addDiskDevice(id, driver string) {
	dv := device{
		driver:  driver,
		devtype: "drive",
		devid:   id,
	}
	q.devices = append(q.devices, dv)
}

// versionCompare compares Qemu version numbers. If the the first argument is
// greater then true is returned, if the second argument is greater
// then versionCompare returns false, otherwise it returns true.
func (q *qemu) versionCompare(v1, v2 string) (bool, error) {
	const formatError = "improperly formatted qemu version %q"

	ver1 := strings.Split(v1, ".")
	ver2 := strings.Split(v2, ".")

	errV1 := fmt.Errorf(formatError, v1)
	errV2 := fmt.Errorf(formatError, v2)

	if len(ver1) < 2 {
		return false, errV1
	}
	if len(ver2) < 2 {
		return false, errV2
	}

	majorV1, err := strconv.Atoi(ver1[0])
	if err != nil {
		return false, err
	}

	minorV1, err := strconv.Atoi(ver1[1])
	if err != nil {
		return false, errV1
	}

	majorV2, err := strconv.Atoi(ver2[0])
	if err != nil {
		return false, err
	}

	minorV2, err := strconv.Atoi(ver2[1])
	if err != nil {
		return false, errV2
	}

	if majorV1 < majorV2 {
		return false, nil
	}

	if majorV1 == majorV2 && minorV1 < minorV2 {
		return false, nil
	}

	return true, nil
}

func (q *qemu) addFlag(flag string) {
	q.flags = append(q.flags, flag)
}

func (q *qemu) addOption(flag, value string) {
	option := fmt.Sprintf("%s %s", flag, value)
	q.flags = append(q.flags, option)
}

// setAccel - trying to set accel.
// Shows Warning messages if can't enable.
// May terminate ops (depends on error, see qemu_errors:qemuAccelWarningMessage for details).
func (q *qemu) setAccel(rconfig *types.RunConfig) {
	var (
		isAdded      bool  = false
		supportedErr error = &errQemuHWAccelDisabledInConfig{errCustom{"Hardware acceleration disabled in config", nil}}
	)

	if rconfig.Accel {
		isAdded, supportedErr = q.addAccel()
	}

	if supportedErr != nil {
		msg, terminate := qemuAccelWarningMessage(supportedErr)
		if msg != "" {
			log.Warn(msg)
		}
		if terminate {
			os.Exit(1)
		}
		if isAdded {
			log.Warn("Anyway, we will try to enable hardware acceleration\n")
		}
	}
}

// addAccel - trying to enable hardware acceleration and check if it is supported.
// In some cases it tries to enable even if it's not supported.
// Returns true in case if it tries to enable, false otherwise.
// Returns err if not supported, nil otherwise.
func (q *qemu) addAccel() (bool, error) {
	// hv_support should not throw errors. When error/log handling is
	// implemented, then errors in detecting hardware acceleration support
	// should be reported. Here we treat any error in detecting hardware
	// acceleration the same as detecting no hardware acceleration - we don't
	// add the flag.

	// Check qemu is installed or not
	if q.isInstalled() == false {
		return false, &errQemuNotInstalled{errCustom{"QEMU not found", nil}}
	}

	const hvfSupportedVersion = "2.12" // https://wiki.qemu.org/ChangeLog/2.12#Host_support
	qemuVersion, err := Version()
	if err != nil {
		return false, &errQemuCannotGetQemuVersion{errCustom{"cannot get QEMU version", err}}
	}

	ok, err := hvSupport()
	if !(ok && err == nil) {
		return false, &errQemuHWAccelNotSupported{errCustom{"Hardware acceleration not supported", err}}
	}

	if lepton.AltGOARCH != "" {
		return false, nil
	}

	if runtime.GOOS == "darwin" {
		if ok, _ := q.versionCompare(qemuVersion, hvfSupportedVersion); ok {
			q.addOption("-accel", "hvf")
			q.addOption("-cpu", "host,-rdtscp")
			return true, nil
		}
		return false, &errQemuHWAccelNotSupported{errCustom{"Hardware acceleration not supported", nil}}
	}

	if runtime.GOOS == "linux" && !isInception() {
		q.addFlag("-machine accel=kvm:tcg")
		q.addOption("-cpu", "host")
		if err := kvmAvailable(); err != nil {
			return true, &errQemuHWAccelNotSupported{errCustom{"Hardware acceleration not supported", err}}
		}
		return true, nil
	}

	if isInception() {
		q.addFlag("--enable-kvm")
	}

	return false, nil
}

func (q *qemu) setConfig(rconfig *types.RunConfig) error {
	if rconfig.GPUs > 0 {
		gpuType := rconfig.GPUType
		if gpuType == "" {
			gpuType = "pci-passthrough"
		}
		if gpuType == "pci-passthrough" {
			if runtime.GOOS != "linux" {
				return fmt.Errorf("GPU passthrough is only supported on Linux")
			}
			// Enable passthrough on PCI devices with class 03 (display).
			if err := q.addVfioPci("03", rconfig.GPUs); err != nil {
				return fmt.Errorf("Error setting up PCI GPU passthrough: %v", err)
			}
		} else {
			return fmt.Errorf("Invalid GPU type '%s'", gpuType)
		}
	}

	// add virtio drive
	q.addDrive("hd0", rconfig.ImageName, "none")

	pciBus := ""
	if isx86() || lepton.AltGOARCH == "amd64" {
		pciBus = "pcie.0"
	}

	usex86 := ArchCheck()

	if usex86 {
		q.addOption("-machine", "q35")

		q.addOption("-device", "pcie-root-port,port=0x10,chassis=1,id=pci.1,bus="+pciBus+",multifunction=on,addr=0x3")
		q.addOption("-device", "pcie-root-port,port=0x11,chassis=2,id=pci.2,bus="+pciBus+",addr=0x3.0x6")
		q.addOption("-device", "pcie-root-port,port=0x12,chassis=3,id=pci.4,bus="+pciBus+",addr=0x3.0x4")

		// FIXME for multiple local tenants
		// x86

		if !isInception() {
			q.addOption("-device", "virtio-scsi-pci,bus=pci.2,addr=0x0,id=scsi0")
		} else {
			q.addOption("-device", "virtio-scsi-pci,id=scsi0,disable-modern=true")
		}

		q.addOption("-device", "scsi-hd,bus=scsi0.0,drive=hd0")

		if !rconfig.Vga {
			q.addOption("-vga", "none")
		}

		q.addOption("-device", "isa-debug-exit")

	} else {
		q.addOption("-machine", "virt")

		q.addOption("-machine", "gic-version=2")

		q.addOption("-kernel", rconfig.Kernel)

		q.addOption("-device", "virtio-blk-pci,drive=hd0")

		q.addFlag("-semihosting")
	}

	if isInception() {
		q.addOption("-kernel", rconfig.Kernel)
		q.addOption("-bios", "/usr/share/qemu/qboot.rom")
	}

	if rconfig.CPUs > 0 {
		q.addOption("-smp", strconv.Itoa(rconfig.CPUs))
	}

	q.addOption("-m", rconfig.Memory)

	q.addOption("-device", "virtio-rng-pci")
	q.addOption("-device", "virtio-balloon")

	disks := 1

	if len(rconfig.VirtfsShares) > 0 {

		for k, v := range rconfig.VirtfsShares {
			mntDir := k
			hostDir := v

			q.addOption("-virtfs", fmt.Sprintf("local,path=%s,mount_tag=%s,security_model=none,multidevs=remap,id=hd%s", hostDir, mntDir, mntDir))
			disks++
		}
	}

	// add mounted volumes
	for _, file := range rconfig.Mounts {
		q.addDrive(fmt.Sprintf("hd%d", disks), file, "none")
		q.addOption("-device", fmt.Sprintf("scsi-hd,bus=scsi0.0,drive=hd%d", disks))
		disks++
	}

	netDevType := "user"
	goos := runtime.GOOS

	ifaceName := ""
	if rconfig.Bridged {
		netDevType = "tap"
		ifaceName = rconfig.TapName
	}

	q.setAccel(rconfig)

	if goos != "freebsd" {
		nics := rconfig.Nics

		if len(nics) > 0 {
			for i := 0; i < len(nics); i++ {
				q.addNetDevice(netDevType, ifaceName, "", rconfig.Ports, rconfig.UDPPorts)
			}
		} else {
			q.addNetDevice(netDevType, ifaceName, "", rconfig.Ports, rconfig.UDPPorts)
		}
	}

	q.addDisplay("none")

	if rconfig.Background {
		q.addSerial("file:/tmp/" + rconfig.InstanceName + ".log")
	} else {
		q.addSerial("stdio")
	}

	if OPSD != "" || MACPKGD != "" {
		q.addOption("-L", "/Applications/qemu.app/Contents/MacOS/")
	}

	if runtime.GOOS == "darwin" && runtime.GOARCH != "arm64" && lepton.AltGOARCH == "" {
		q.addOption("-cpu", "max,-rdtscp")
	} else if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" && lepton.AltGOARCH == "" {
		q.addOption("-cpu", "host")
	} else {
		if !isInception() {
			q.addOption("-cpu", "max")
		}
	}

	if rconfig.GdbPort > 0 {
		gdbProtoStr := fmt.Sprintf("tcp::%d", rconfig.GdbPort)
		q.addOption("-gdb", gdbProtoStr)
	}

	if rconfig.AtExit != "" {
		q.atExitHook = rconfig.AtExit
	}

	if rconfig.Debug {
		gdbPort := "1234"
		if rconfig.GdbPort != 0 {
			gdbPort = strconv.Itoa(rconfig.GdbPort)
		}

		q.addOption("-s", "-S")
		fmt.Printf("Waiting for gdb connection. Connect to qemu through \"(gdb) target remote localhost:%s\"\n", gdbPort)
		fmt.Println("See further instructions in https://nanovms.gitbook.io/ops/debugging")
	}

	return nil
}

func (q *qemu) isInstalled() bool {
	qemuCommand := qemuBaseCommand()
	if filepath.Base(qemuCommand) == qemuCommand {
		lp, err := exec.LookPath(qemuCommand)
		if err != nil {
			return false
		}
		qemuCommand = lp
	}

	fi, err := os.Stat(qemuCommand)
	if err != nil || fi.IsDir() {
		return false
	}

	// Can any (Owner, Group or Other) execute the file
	return fi.Mode().Perm()&0111 != 0
}

func (q *qemu) Args(rconfig *types.RunConfig) ([]string, error) {
	err := q.setConfig(rconfig)
	if err != nil {
		return nil, err
	}
	args := []string{}

	// pci bus needs to be declared before devices using them
	for _, flag := range q.flags {
		args = append(args, flag)
	}

	for _, drive := range q.drives {
		args = append(args, drive.String())
	}

	for _, device := range q.devices {
		args = append(args, device.String())
	}

	for _, iface := range q.ifaces {
		args = append(args, iface.String())
	}

	args = append(args, q.display.String())
	args = append(args, q.serial.String())

	// for now let's make this optional
	if rconfig.QMP {
		args = append(args, "-qmp tcp:localhost:"+rconfig.Mgmt+",server,wait=off")
	}

	if isInception() {
		args = append(args, "-device pci-bridge,bus=pcie.0,id=pci-bridge-0,chassis_nr=1,shpc=off,addr=6,io-reserve=4k,mem-reserve=1m,pref64-reserve=1m")
	}

	// The returned args must tokenized by whitespace
	return strings.Fields(strings.Join(args, " ")), nil
}

func (q *qemu) PID() (string, error) {
	if q.cmd == nil || q.cmd.Process == nil {
		return "", errors.New("No process running")
	}

	return strconv.Itoa(q.cmd.Process.Pid), nil
}

// Randomly generate Bytes for mac address
func generateMac() string {
	octets := make([]byte, 6)
	_, err := rand.Read(octets)
	if err != nil {
		fmt.Printf("Failed generate Mac, error is: %s", err)
		os.Exit(1)
	}
	octets[0] |= 2
	octets[0] &= 0xFE //mask most sig bit for unicast at layer 2
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		octets[0], octets[1], octets[2], octets[3], octets[4], octets[5])
}

// Version gives the version of qemu running locally.
func Version() (string, error) {
	versionData, err := exec.Command(qemuBaseCommand(), "--version").Output()
	if err != nil {
		return "", &errQemuCannotExecute{errCustom{"cannot execute QEMU", err}}
	}
	return parseQemuVersion(versionData), nil
}

func parseQemuVersion(data []byte) string {
	rgx := regexp.MustCompile("[0-9]+\\.[0-9]+\\.[0-9]+")
	return rgx.FindString(string(data))
}

// kvmAvailable returns nil if the current user have read and write access to /dev/kvm
// or error in other case
func kvmAvailable() error {
	return syscall.Access("/dev/kvm", unix.R_OK|unix.W_OK)
}

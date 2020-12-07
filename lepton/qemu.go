package lepton

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const qemuBaseCommand = "qemu-system-x86_64"

type drive struct {
	path   string
	format string
	iftype string
	index  string
	ID     string
}

type device struct {
	driver  string
	devtype string
	mac     string
	devid   string
}

type netdev struct {
	nettype    string
	id         string
	ifname     string
	script     string
	downscript string
	hports     []portfwd
}

type portfwd struct {
	port  string
	proto string
}

type display struct {
	disptype string
}

type serial struct {
	serialtype string
}

type qemu struct {
	cmd     *exec.Cmd
	drives  []drive
	devices []device
	ifaces  []netdev
	display display
	serial  serial
	flags   []string
}

func (d display) String() string {
	return fmt.Sprintf("-display %s", d.disptype)
}

func (s serial) String() string {
	return fmt.Sprintf("-serial %s", s.serialtype)
}

func (d drive) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-drive file=%s,format=%s", d.path, d.format))
	if len(d.index) > 0 {
		sb.WriteString(fmt.Sprintf(",index=%s", d.index))
	}
	if len(d.iftype) > 0 {
		sb.WriteString(fmt.Sprintf(",if=%s", d.iftype))
	}
	if len(d.ID) > 0 {
		sb.WriteString(fmt.Sprintf(",id=%s", d.ID))
	}
	return sb.String()
}

func (dv device) String() string {
	var sb strings.Builder

	// simple pci net hack -- FIXME
	if dv.driver == "virtio-net" {
		sb.WriteString(fmt.Sprintf("-device %s,bus=pci.3,addr=0x0,%s=%s", dv.driver, dv.devtype, dv.devid))
	} else {
		sb.WriteString(fmt.Sprintf("-device %s,%s=%s", dv.driver, dv.devtype, dv.devid))
	}

	if len(dv.mac) > 0 {
		sb.WriteString(fmt.Sprintf(",mac=%s", dv.mac))
	}
	return sb.String()
}

func (nd netdev) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-netdev %s,id=%s", nd.nettype, nd.id))
	if len(nd.ifname) > 0 {
		sb.WriteString(fmt.Sprintf(",ifname=%s", nd.ifname))
	}
	if nd.nettype != "user" {
		if len(nd.script) > 0 {
			sb.WriteString(fmt.Sprintf(",script=%s", nd.script))
		} else {
			sb.WriteString(",script=no")
		}
		if len(nd.downscript) > 0 {
			sb.WriteString(fmt.Sprintf(",downscript=%s", nd.downscript))
		} else {
			sb.WriteString(",downscript=no")
		}
	}
	for _, hport := range nd.hports {
		sb.WriteString(fmt.Sprintf(",%s", hport))
	}
	return sb.String()
}

func (pf portfwd) String() string {
	fromPort := pf.port
	toPort := pf.port

	if strings.Contains(fromPort, "-") {
		rangeParts := strings.Split(fromPort, "-")
		fromPort = rangeParts[0]
		toPort = rangeParts[1]
	}

	return fmt.Sprintf("hostfwd=%s::%v-:%v", pf.proto, fromPort, toPort)
}

func (q *qemu) Stop() {
	if q.cmd != nil {
		if err := q.cmd.Process.Kill(); err != nil {
			fmt.Println(err)
		}

		// do not print errors as the command could be started with Run()
		q.cmd.Wait()
	}
}

func logv(rconfig *RunConfig, msg string) {
	if rconfig.Verbose {
		fmt.Println(msg)
	}
}

func (q *qemu) Command(rconfig *RunConfig) *exec.Cmd {
	args := q.Args(rconfig)
	logv(rconfig, qemuBaseCommand+" "+strings.Join(args, " "))
	q.cmd = exec.Command(qemuBaseCommand, args...)

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

func (q *qemu) Start(rconfig *RunConfig) error {
	if q.cmd == nil {
		q.Command(rconfig)
		q.cmd.Stdout = os.Stdout
		q.cmd.Stderr = os.Stderr
	}

	if rconfig.OnPrem {
		err := q.cmd.Start()
		if err != nil {
			fmt.Println(err)
		}

		pid := strconv.Itoa(q.cmd.Process.Pid)
		opshome := GetOpsHome()
		instances := path.Join(opshome, "instances")

		base := path.Base(rconfig.Imagename)
		sbase := strings.Split(base, ".")

		i := instance{
			Image: sbase[0],
			Ports: rconfig.Ports,
		}

		d1, err := json.Marshal(i)
		if err != nil {
			fmt.Println(err)
		}

		err = ioutil.WriteFile(instances+"/"+pid, d1, 0644)
		if err != nil {
			fmt.Println(err)
		}

	} else {

		if err := q.cmd.Run(); err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

func (q *qemu) addDrive(id, image, ifaceType string) {
	drv := drive{
		path:   image,
		format: "raw",
		iftype: ifaceType,
		ID:     id,
	}
	q.drives = append(q.drives, drv)
}

func (q *qemu) addDisplay(dispType string) {
	q.display = display{disptype: dispType}
}

func (q *qemu) addSerial(serialType string) {
	q.serial = serial{serialtype: serialType}
}

// addDevice adds a device to the qemu for rendering to string arguments. If the
// devType is "user" then the ifaceName is ignored and host forward ports are
// added. If the mac address is empty then a random mac address is chosen.
// Backend interface are created for each device and their ids are auto
// incremented.
func (q *qemu) addNetDevice(devType, ifaceName, mac string, hostPorts []string, udp bool) {
	id := fmt.Sprintf("n%d", len(q.ifaces))
	dv := device{
		driver:  "virtio-net",
		devtype: "netdev",
		devid:   id,
	}
	ndv := netdev{
		nettype: devType,
		id:      id,
	}

	if devType != "user" {
		if mac == "" {
			dv.mac = generateMac()
		}
		ndv.ifname = ifaceName
	} else {
		for _, p := range hostPorts {
			ndv.hports = append(ndv.hports, portfwd{port: p, proto: "tcp"})

			if udp {
				ndv.hports = append(ndv.hports, portfwd{port: p, proto: "udp"})
			}
		}
	}

	q.devices = append(q.devices, dv)
	q.ifaces = append(q.ifaces, ndv)
}

func (q *qemu) addDiskDevice(id, driver string) {
	dv := device{
		driver:  driver,
		devtype: "drive",
		devid:   id,
	}
	q.devices = append(q.devices, dv)
}

// Randomly generate Bytes for mac address
func generateMac() string {
	octets := make([]byte, 6)
	_, err := rand.Read(octets)
	if err != nil {
		panic(err)
	}
	octets[0] |= 2
	octets[0] &= 0xFE //mask most sig bit for unicast at layer 2
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		octets[0], octets[1], octets[2], octets[3], octets[4], octets[5])
}

// QemuVersion gives the version of qemu running locally.
func QemuVersion() (string, error) {
	versionData, err := exec.Command(qemuBaseCommand, "--version").Output()
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
func (q *qemu) setAccel(rconfig *RunConfig) {
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
			fmt.Println(msg)
		}
		if terminate {
			os.Exit(1)
		}
		if isAdded {
			fmt.Printf(WarningColor, "Anyway, we will try to enable hardware acceleration\n")
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
	qemuVersion, err := QemuVersion()
	if err != nil {
		return false, &errQemuCannotGetQemuVersion{errCustom{"cannot get QEMU version", err}}
	}

	ok, err := hvSupport()
	if !(ok && err == nil) {
		return false, &errQemuHWAccelNotSupported{errCustom{"Hardware acceleration not supported", err}}
	}

	if runtime.GOOS == "darwin" {
		if ok, _ := q.versionCompare(qemuVersion, hvfSupportedVersion); ok {
			q.addOption("-accel", "hvf")
			q.addOption("-cpu", "host")
			return true, nil
		}
		return false, &errQemuHWAccelNotSupported{errCustom{"Hardware acceleration not supported", nil}}
	}

	if runtime.GOOS == "linux" {
		q.addFlag("-machine accel=kvm:tcg")
		q.addOption("-cpu", "host")
		if err := kvmAvailable(); err != nil {
			return true, &errQemuHWAccelNotSupported{errCustom{"Hardware acceleration not supported", err}}
		}
		return true, nil
	}

	return false, nil
}

func (q *qemu) setConfig(rconfig *RunConfig) {
	// add virtio drive
	q.addDrive("hd0", rconfig.Imagename, "none")

	pciBus := "pcie.0"

	// pcie root ports need to come before virtio/scsi devices
	q.addOption("-machine", "q35")
	q.addOption("-device", "pcie-root-port,port=0x10,chassis=1,id=pci.1,bus="+pciBus+",multifunction=on,addr=0x3")
	q.addOption("-device", "pcie-root-port,port=0x11,chassis=2,id=pci.2,bus="+pciBus+",addr=0x3.0x1")
	q.addOption("-device", "pcie-root-port,port=0x12,chassis=3,id=pci.3,bus="+pciBus+",addr=0x3.0x2")

	// FIXME for multiple local tenants
	q.addOption("-device", "virtio-scsi-pci,bus=pci.2,addr=0x0,id=scsi0")
	q.addOption("-device", "scsi-hd,bus=scsi0.0,drive=hd0")

	// add mounted volumes
	for n, file := range rconfig.Mounts {
		q.addDrive(fmt.Sprintf("hd%d", n+1), file, "none")
		q.addOption("-device", fmt.Sprintf("scsi-hd,bus=scsi0.0,drive=hd%d", n+1))
	}

	netDevType := "user"
	ifaceName := ""
	if rconfig.Bridged {
		netDevType = "tap"
		ifaceName = rconfig.TapName
	}

	q.setAccel(rconfig)

	q.addNetDevice(netDevType, ifaceName, "", rconfig.Ports, rconfig.UDP)
	q.addDisplay("none")

	if rconfig.OnPrem {
		q.addSerial("file:/tmp/" + rconfig.BaseName + ".log")
	} else {
		q.addSerial("stdio")
	}

	q.addFlag("-no-reboot")
	q.addOption("-cpu", "max")

	if rconfig.CPUs > 0 {
		q.addOption("-smp", strconv.Itoa(rconfig.CPUs))
	}

	// we could perhaps cascade for different versions of qemu here but
	// I think everyone should have this
	q.addOption("-machine", "q35")

	q.addOption("-device", "isa-debug-exit")
	q.addOption("-m", rconfig.Memory)

	if rconfig.GdbPort > 0 {
		gdbProtoStr := fmt.Sprintf("tcp::%d", rconfig.GdbPort)
		q.addOption("-gdb", gdbProtoStr)
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
}

func (q *qemu) isInstalled() bool {
	qemuCommand := qemuBaseCommand
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

func (q *qemu) Args(rconfig *RunConfig) []string {
	q.setConfig(rconfig)
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

	// The returned args must tokenized by whitespace
	return strings.Fields(strings.Join(args, " "))
}

func newQemu() Hypervisor {
	return &qemu{}
}

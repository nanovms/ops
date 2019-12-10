package lepton

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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
	port  int
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
	sb.WriteString(fmt.Sprintf("-device %s,%s=%s", dv.driver, dv.devtype, dv.devid))
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
	return fmt.Sprintf("hostfwd=%s::%v-:%v", pf.proto, pf.port, pf.port)
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

	if err := q.cmd.Run(); err != nil {
		fmt.Println(err)
		return err
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
func (q *qemu) addNetDevice(devType, ifaceName, mac string, hostPorts []int) {
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
		return "", err
	}
	return parseQemuVersion(versionData), nil
}

func parseQemuVersion(data []byte) string {
	rgx := regexp.MustCompile("[0-9]+\\.[0-9]+\\.[0-9]+")
	return rgx.FindString(string(data))
}

// kvmGroupPermission returns true if the current user is part of the kvm qroup,
// false if she is not, or an error. or an error.
func kvmGroupPermissions() (bool, error) {
	currentUser, err := user.Current()
	if err != nil {
		return false, err
	}
	if currentUser.Uid == "0" {
		return true, nil
	}
	return groupPermissions(currentUser, "kvm")
}

// groupPermissions returns true if the user is in the group specified by the
// argument string, false if she is not, or an error.
func groupPermissions(u *user.User, groupName string) (bool, error) {
	gids, err := u.GroupIds()
	if err != nil {
		return false, err
	}
	g, err := user.LookupGroup(groupName)
	if err != nil {
		return false, err
	}
	for _, gid := range gids {
		if g.Gid == gid {
			return true, nil
		}
	}

	return false, nil
}

// versionCompare compares Qemu version numbers. If the the first argument is
// greater then true is returned, if the second argument is greater
// then versionCompare returns false, otherwise it returns true.
func (q *qemu) versionCompare(v1, v2 string) (bool, error) {
	const formatError = "improperly formated qemu version %q"

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

func (q *qemu) addAccel() {
	// hv_support should not throw errors. When error/log handling is
	// implemented, then errors in detecting hardware acceleration support
	// should be reported. Here we treat any error in detecting hardware
	// acceleration the same as detecting no hardware acceleration - we don't
	// add the flag.

	const hvfSupportedVersion = "2.12" // https://wiki.qemu.org/ChangeLog/2.12#Host_support
	qemuVersion, err := QemuVersion()
	if err != nil {
		return
	}

	ok, err := hvSupport()
	if !(ok && err == nil) {
		return
	}

	if runtime.GOOS == "darwin" {
		if ok, _ := q.versionCompare(qemuVersion, hvfSupportedVersion); ok {
			q.addOption("-accel", "hvf")
			q.addOption("-cpu", "host")
			return
		}
	}
	if runtime.GOOS == "linux" {
		ok, err := kvmGroupPermissions()
		if !(ok && err == nil) {
			fmt.Printf("You specified hardware acceleration but you don't have rights.\n" +
				"Try adding yourself to the kvm group: `sudo adduser $user kvm`\n" +
				"You'll need to re login for this to take affect.\n")
			os.Exit(1)
		}
		q.addFlag("-enable-kvm")
		q.addOption("-cpu", "host")
		return
	}
}

func (q *qemu) setConfig(rconfig *RunConfig) {
	// add virtio drive
	q.addDrive("hd0", rconfig.Imagename, "none")
	q.addDiskDevice("hd0", "virtio-blk")

	netDevType := "user"
	ifaceName := ""
	if rconfig.Bridged {
		netDevType = "tap"
		ifaceName = rconfig.TapName
	}
	if rconfig.Accel || rconfig.Bridged {
		q.addAccel()
	}

	q.addNetDevice(netDevType, ifaceName, "", rconfig.Ports)
	q.addDisplay("none")
	q.addSerial("stdio")
	q.addFlag("-nodefaults")
	q.addFlag("-no-reboot")
	q.addOption("-device", "isa-debug-exit")
	q.addOption("-m", rconfig.Memory)

	if rconfig.GdbPort > 0 {
		gdbProtoStr := fmt.Sprintf("tcp::%d", rconfig.GdbPort)
		q.addOption("-gdb", gdbProtoStr)
	}
}

func (q *qemu) Args(rconfig *RunConfig) []string {
	q.setConfig(rconfig)
	args := []string{}

	for _, drive := range q.drives {
		args = append(args, drive.String())
	}

	for _, device := range q.devices {
		args = append(args, device.String())
	}

	for _, iface := range q.ifaces {
		args = append(args, iface.String())
	}

	for _, flag := range q.flags {
		args = append(args, flag)
	}

	args = append(args, q.display.String())
	args = append(args, q.serial.String())

	// The returned args must tokenized by whitespace
	return strings.Fields(strings.Join(args, " "))
}

func newQemu() Hypervisor {
	return &qemu{}
}

package lepton

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

const qemuBaseCommand = "qemu-system-x86_64"

type drive struct {
	path   string
	format string
	iftype string
	index  string
}

type device struct {
	driver   string
	mac      string
	netdevid string
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
	return sb.String()
}

func (dv device) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-device %s,netdev=%s", dv.driver, dv.netdevid))
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
	}
}

func logv(rconfig *RunConfig, msg string) {
	if rconfig.Verbose {
		fmt.Println(msg)
	}
}

func (q *qemu) Command(rconfig *RunConfig) *exec.Cmd {
	args := q.Args(rconfig)
	q.cmd = exec.Command(qemuBaseCommand, args...)
	return q.cmd
}

func (q *qemu) Start(rconfig *RunConfig) error {
	args := q.Args(rconfig)
	logv(rconfig, qemuBaseCommand+" "+strings.Join(args, " "))

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

	q.cmd = exec.Command(qemuBaseCommand, args...)
	q.cmd.Stdout = os.Stdout
	q.cmd.Stderr = os.Stderr

	if err := q.cmd.Run(); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (q *qemu) addDrive(image, ifaceType string) {
	drv := drive{
		path:   image,
		format: "raw",
		iftype: ifaceType,
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
func (q *qemu) addDevice(devType, ifaceName, mac string, hostPorts []int) {
	id := fmt.Sprintf("n%d", len(q.ifaces))
	dv := device{
		driver:   "virtio-net",
		netdevid: id,
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

func (q *qemu) addFlag(flag string) {
	q.flags = append(q.flags, flag)
}

func (q *qemu) addOption(flag, value string) {
	option := fmt.Sprintf("%s %s", flag, value)
	q.flags = append(q.flags, option)
}

func (q *qemu) setConfig(rconfig *RunConfig) {
	// add virtio drive
	q.addDrive(rconfig.Imagename, "virtio")
	devType := "user"
	ifaceName := ""
	if rconfig.Bridged {
		devType = "tap"
		ifaceName = rconfig.TapName
		if runtime.GOOS == "darwin" {
			q.addFlag("-enable-hax")
		} else {
			q.addFlag("-enable-kvm")
		}
	}
	q.addDevice(devType, ifaceName, "", rconfig.Ports)
	q.addDisplay("none")
	q.addSerial("stdio")
	q.addFlag("-nodefaults")
	q.addFlag("-no-reboot")
	q.addOption("-device", "isa-debug-exit")
	q.addOption("-m", rconfig.Memory)
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

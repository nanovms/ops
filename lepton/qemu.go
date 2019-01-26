package lepton

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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
		q.cmd.Process.Kill()
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
	q.cmd = exec.Command(qemuBaseCommand, args...)
	q.cmd.Stdout = os.Stdout
	q.cmd.Stdin = os.Stdin
	q.cmd.Stderr = os.Stderr

	if err := q.cmd.Run(); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (q *qemu) Args(rconfig *RunConfig) []string {
	boot := drive{path: "image", format: "raw", index: "0"}
	storage := drive{path: "image", format: "raw", iftype: "virtio"}
	dev := device{driver: "virtio-net", mac: "7e:b8:7e:87:4a:ea", netdevid: "n0"}
	ndev := netdev{nettype: "tap", id: "n0", ifname: "tap0"}
	display := display{disptype: "none"}
	serial := serial{serialtype: "stdio"}
	if !rconfig.Bridged {
		hps := []portfwd{}
		for _, p := range rconfig.Ports {
			hps = append(hps, portfwd{port: p, proto: "tcp"})
		}
		dev = device{driver: "virtio-net", netdevid: "n0"}
		ndev = netdev{nettype: "user", id: "n0", hports: hps}
	}
	args := []string{
		boot.String(),
		display.String(),
		serial.String(),
		"-nodefaults",
		"-no-reboot",
		"-m", rconfig.Memory,
		"-device", "isa-debug-exit",
		storage.String(),
		dev.String(),
		ndev.String(),
	}
	if rconfig.Bridged {
		args = append(args, "-enable-kvm")
	}
	// The returned args must tokenized by whitespace
	return strings.Fields(strings.Join(args, " "))
}

func newQemu() Hypervisor {
	return &qemu{}
}

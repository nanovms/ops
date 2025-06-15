package qemu

import (
	"fmt"
	"strings"

	"github.com/nanovms/ops/lepton"
)

type drive struct {
	path   string
	format string
	iftype string
	index  string
	ID     string
}

func (d drive) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-drive file=%s", d.path))
	if len(d.format) > 0 {
		sb.WriteString(fmt.Sprintf(",format=%s", d.format))
	}
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

type device struct {
	driver  string
	devtype string
	mac     string
	devid   string
	addr    string
}

// ArchCheck returns true if the user is on or wants to use amd64
// otherwise it returns false.
func ArchCheck() bool {
	usex86 := true
	if (isx86() && lepton.AltGOARCH == "") || lepton.AltGOARCH == "amd64" {
		usex86 = true
	}

	if (!isx86() && lepton.AltGOARCH == "") || lepton.AltGOARCH == "arm64" {
		usex86 = false
	}

	return usex86
}

func (dv device) String() string {
	var sb strings.Builder

	if dv.addr == "" {
		dv.addr = "0x0"
	}

	// simple pci net hack -- FIXME
	if dv.driver == "virtio-net" {

		usex86 := ArchCheck()

		if usex86 {
			if isInception() {
				sb.WriteString(fmt.Sprintf("-device %s,addr=5,%s=%s", dv.driver, dv.devtype, dv.devid))
			} else {
				sb.WriteString(fmt.Sprintf("-device %s,bus=pci.3,addr=%s,%s=%s", dv.driver, dv.addr, dv.devtype, dv.devid))
			}

		} else {
			sb.WriteString(fmt.Sprintf("-device %s,%s=%s", dv.driver, dv.devtype, dv.devid))
		}

	} else if dv.driver == "virtio-net-pci" {
		sb.WriteString(fmt.Sprintf("-device %s,%s", dv.driver, dv.devtype))

	} else {
		sb.WriteString(fmt.Sprintf("-device %s,%s=%s", dv.driver, dv.devtype, dv.devid))
	}

	if len(dv.mac) > 0 {
		sb.WriteString(fmt.Sprintf(",mac=%s", dv.mac))
	}
	return sb.String()
}

type netdev struct {
	nettype    string
	id         string
	ifname     string
	script     string
	downscript string
	hports     []portfwd
}

func (nd netdev) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-netdev %s,id=%s", nd.nettype, nd.id))
	if len(nd.ifname) > 0 {
		sb.WriteString(fmt.Sprintf(",ifname=%s", nd.ifname))
	}
	if nd.nettype != "user" && nd.nettype != "vmnet-bridged" {
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

type portfwd struct {
	port  string
	proto string
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

type display struct {
	disptype string
}

func (d display) String() string {
	return fmt.Sprintf("-display %s", d.disptype)
}

type serial struct {
	serialtype string
}

func (s serial) String() string {
	return fmt.Sprintf("-serial %s", s.serialtype)
}

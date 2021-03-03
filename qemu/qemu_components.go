package qemu

import (
	"fmt"
	"strings"
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

type device struct {
	driver  string
	devtype string
	mac     string
	devid   string
}

func (dv device) String() string {
	var sb strings.Builder

	// simple pci net hack -- FIXME
	if dv.driver == "virtio-net" {

		if isx86() {
			sb.WriteString(fmt.Sprintf("-device %s,bus=pci.3,addr=0x0,%s=%s", dv.driver, dv.devtype, dv.devid))
		} else {
			sb.WriteString(fmt.Sprintf("-device %s,%s=%s", dv.driver, dv.devtype, dv.devid))
		}

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

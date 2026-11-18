package qemu

import (
	"fmt"
	"strings"
)

type device struct {
	arch    string
	driver  string
	devtype string
	mac     string
	devid   string
	addr    string
}

func (dv device) String() string {
	var sb strings.Builder

	if dv.addr == "" {
		dv.addr = "0x0"
	}

	// simple pci net hack -- FIXME
	if dv.driver == "virtio-net" {

		usex86 := ArchCheck(dv.arch)

		if usex86 {
			if isInception() {
				sb.WriteString(fmt.Sprintf("-device %s,addr=5,%s=%s", dv.driver, dv.devtype, dv.devid))
			} else {
				sb.WriteString(fmt.Sprintf("-device %s,bus=pci.4,addr=%s,%s=%s", dv.driver, dv.addr, dv.devtype, dv.devid))
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

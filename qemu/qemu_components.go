package qemu

import (
	"fmt"
	"strings"
)

// ArchCheck returns true if the user is on or wants to use amd64
// otherwise it returns false.
//
// FIXME: rename to use86
func ArchCheck(arch string) bool {
	if arch == "arm64" {
		return false
	}

	return true
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

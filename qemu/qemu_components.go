package qemu

import (
	"fmt"
	"strings"

	"github.com/nanovms/ops/lepton"
)

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

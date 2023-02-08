package onprem

import (
	"strings"
)

// for now only assumes a single interface
// TOOD: get from protos
type instance struct {
	Instance  string   `json:"instance"`
	Image     string   `json:"image"`
	Ports     []string `json:"ports"`
	Bridged   bool     `json:"bridged"`
	PrivateIP string   `json:"private_ip"` // assume only loopback unless bridged is set, only set if bridged is set
	Mac       string   `json:"mac"`
	Pid       string   `json:"pid"`
}

func (in *instance) portList() string {
	s := ""
	for i := 0; i < len(in.Ports); i++ {
		s += in.Ports[i] + ", "
	}

	return strings.TrimRight(s, ", ")
}

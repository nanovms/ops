package onprem

import (
	"strings"
)

// for now only assumes a single interface
// TOOD: get from protos
// instance stores metadata for on-prem instances and contains
// different/extra fields than what one would find in cloudInstance
type instance struct {
	Instance  string   `json:"instance"` // instance name
	Image     string   `json:"image"`
	Ports     []string `json:"ports"`
	Bridged   bool     `json:"bridged"`
	PrivateIP string   `json:"private_ip"` // assume only loopback unless bridged is set, only set if bridged is set
	Mac       string   `json:"mac"`
	Pid       string   `json:"pid"`
	Mgmt      string   `json:"mgmt"`
}

func (in *instance) portList() string {
	s := ""
	for i := 0; i < len(in.Ports); i++ {
		s += in.Ports[i] + ", "
	}

	return strings.TrimRight(s, ", ")
}

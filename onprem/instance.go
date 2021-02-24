package onprem

import (
	"strings"
)

type instance struct {
	Image string   `json:"image"`
	Ports []string `json:"ports"`
}

func (in *instance) portList() string {
	s := ""
	for i := 0; i < len(in.Ports); i++ {
		s += in.Ports[i] + ", "
	}

	return strings.TrimRight(s, ", ")
}

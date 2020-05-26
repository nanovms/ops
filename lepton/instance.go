package lepton

import (
	"strconv"
	"strings"
)

type instance struct {
	Image string `json:"image"`
	Ports []int  `json:"ports"`
}

func (in *instance) portList() string {
	s := ""
	for i := 0; i < len(in.Ports); i++ {
		s += strconv.Itoa(in.Ports[i]) + ", "
	}

	return strings.TrimRight(s, ", ")
}

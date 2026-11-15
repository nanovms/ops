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

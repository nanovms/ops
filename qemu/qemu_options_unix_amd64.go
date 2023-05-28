//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package qemu

var hypervisors = map[string]func() Hypervisor{
	"qemu-system-x86_64": newQemu,
}

func isx86() bool {
	return true
}

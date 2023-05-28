//go:build linux || darwin
// +build linux darwin

package qemu

var hypervisors = map[string]func() Hypervisor{
	"qemu-system-aarch64": newQemu,
}

func isx86() bool {
	return false
}

//go:build linux || darwin
// +build linux darwin

package qemu

const qemuBaseCommand = "qemu-system-aarch64"

var hypervisors = map[string]func() Hypervisor{
	"qemu-system-aarch64": newQemu,
}

func isx86() bool {
	return false
}

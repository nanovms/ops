// +build linux darwin

package qemu

const BaseCommand = "qemu-system-x86_64"

var hypervisors = map[string]func() Hypervisor{
	"qemu-system-x86_64": newQemu,
}

func isx86() bool {
	return true
}

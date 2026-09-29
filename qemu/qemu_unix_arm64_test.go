package qemu

import "testing"

func TestStringDevice(t *testing.T) {
	testDevice := &device{driver: "virtio-net",
		mac:     "7e:b8:7e:87:4a:ea",
		devtype: "netdev",
		devid:   "n0"}

	expected := "-device virtio-net,netdev=n0,mac=7e:b8:7e:87:4a:ea"
	checkQemuString(testDevice, expected, t)
}

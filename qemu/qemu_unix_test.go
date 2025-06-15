//go:build linux || darwin
// +build linux darwin

package qemu

import (
	. "fmt"
	"reflect"
	"testing"
)

func TestStringDriveWithIndex(t *testing.T) {
	testDrive := &drive{path: "image", format: "raw", index: "0"}
	expected := "-drive file=image,format=raw,index=0"
	checkQemuString(testDrive, expected, t)
}

func TestStringDriveWithIfType(t *testing.T) {
	testDrive := &drive{path: "image", format: "raw", iftype: "virtio"}
	expected := "-drive file=image,format=raw,if=virtio"
	checkQemuString(testDrive, expected, t)
}

func TestStringDevice(t *testing.T) {
	testDevice := &device{driver: "virtio-net",
		mac:     "7e:b8:7e:87:4a:ea",
		devtype: "netdev",
		devid:   "n0"}

	expected := "-device virtio-net,netdev=n0,mac=7e:b8:7e:87:4a:ea"
	checkQemuString(testDevice, expected, t)
}

func TestStringNetDev(t *testing.T) {
	testNetDev := &netdev{
		nettype:    "tap",
		id:         "n0",
		ifname:     "tap0",
		script:     "no",
		downscript: "no",
	}
	expected := "-netdev tap,id=n0,ifname=tap0,script=no,downscript=no"
	checkQemuString(testNetDev, expected, t)
}

func TestStringNetDevWithHostPortForwarding(t *testing.T) {
	testHostPorts := []portfwd{{proto: "tcp", port: "80"}, {proto: "tcp", port: "443"}}
	testNetDev := &netdev{nettype: "tap", id: "n0", hports: testHostPorts}
	expected := "-netdev tap,id=n0,script=no,downscript=no,hostfwd=tcp::80-:80,hostfwd=tcp::443-:443"
	checkQemuString(testNetDev, expected, t)
}

func TestStringNetDevWithTypeUser(t *testing.T) {
	// The 'downscript' and 'script' parameters are not valid for 'user'
	// device type so we don't render them to the string in that case even
	// if they are populated in the type.
	testHostPorts := []portfwd{{proto: "tcp", port: "80"}, {proto: "tcp", port: "443"}}
	testNetDev := &netdev{nettype: "user", id: "n0", downscript: "no", script: "no", hports: testHostPorts}
	expected := "-netdev user,id=n0,hostfwd=tcp::80-:80,hostfwd=tcp::443-:443"
	checkQemuString(testNetDev, expected, t)
}

func TestStringDisplay(t *testing.T) {
	testDisplay := &display{disptype: "none"}
	expected := "-display none"
	checkQemuString(testDisplay, expected, t)
}

func TestStringSerial(t *testing.T) {
	testSerial := &serial{serialtype: "stdio"}
	expected := "-serial stdio"
	checkQemuString(testSerial, expected, t)
}

func TestRandomMacGen(t *testing.T) {
	q := qemu{}
	q.addNetDevice("tap", "tap0", "", []string{}, []string{})
	if len(q.devices[0].mac) == 0 {
		t.Errorf("No RandomMac was assigned %s", q.devices[0].mac)
	}
}

func TestAddNetDevices(t *testing.T) {

	t.Run("should add a port forward per tcp port", func(t *testing.T) {
		q := qemu{}
		q.addNetDevice("user", "", "", []string{"80", "8080", "9000"}, []string{})

		want := []portfwd{
			{port: "80", proto: "tcp"},
			{port: "8080", proto: "tcp"},
			{port: "9000", proto: "tcp"},
		}
		got := q.ifaces[0].hports

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v,want %v", got, want)
		}
	})

	t.Run("should add a port forward range", func(t *testing.T) {
		q := qemu{}
		q.addNetDevice("user", "", "", []string{"80-9000"}, []string{})

		want := []portfwd{
			{port: "80-9000", proto: "tcp"},
		}
		got := q.ifaces[0].hports

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v,want %v", got, want)
		}
	})
}

func TestQemuVersion(t *testing.T) {
	testData := `
QEMU emulator version 2.11.1(Debian 1:2.8+dfsg-6+deb9u5)
Copyright (c) 2003-2016 Fabrice Bellard and the QEMU Project developers
`
	actual := parseQemuVersion([]byte(testData))
	expected := "2.11.1"
	if actual != expected {
		t.Errorf("Parsed Qemu version %q not %q", actual, expected)
	}
}

func TestVersionCompareGreater(t *testing.T) {
	q := qemu{}
	v1 := "2.19"
	v2 := "2.7"
	res, err := q.versionCompare(v1, v2)
	if err != nil {
		t.Error(err)
	}
	if !res {
		t.Errorf("%q should be a greater version than %q", v1, v2)
	}
}

func TestVersionCompareEqual(t *testing.T) {
	q := qemu{}
	v1 := "2.19"
	v2 := "2.19"
	res, err := q.versionCompare(v1, v2)
	if err != nil {
		t.Error(err)
	}
	if !res {
		t.Errorf("%q should be a greater version than %q", v1, v2)
	}
}

func TestVersionCompareLess(t *testing.T) {
	q := qemu{}
	v1 := "2.7"
	v2 := "2.8"
	res, err := q.versionCompare(v1, v2)
	if err != nil {
		t.Error(err)
	}
	if res {
		t.Errorf("%q should be a prior version to %q", v1, v2)
	}
}

func TestVersionCompareMalformat(t *testing.T) {
	q := qemu{}
	v1 := "2dd.7x"
	v2 := "2.8"
	_, err := q.versionCompare(v1, v2)
	if err.Error() != `strconv.Atoi: parsing "2dd": invalid syntax` {
		t.Errorf("Did not detect improperly formatted error %q", v1)
	}
}

func checkQemuString(qr Stringer, expected string, t *testing.T) {
	actual := qr.String()
	if expected != actual {
		t.Errorf("Rendered string %q not %q", actual, expected)
	}
}

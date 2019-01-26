package lepton

import (
	. "fmt"
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
	testDevice := &device{driver: "virtio-net", mac: "7e:b8:7e:87:4a:ea", netdevid: "n0"}
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
	testHostPorts := []portfwd{{proto: "tcp", port: 80}, {proto: "tcp", port: 443}}
	testNetDev := &netdev{nettype: "tap", id: "n0", hports: testHostPorts}
	expected := "-netdev tap,id=n0,script=no,downscript=no,hostfwd=tcp::80-:80,hostfwd=tcp::443-:443"
	checkQemuString(testNetDev, expected, t)
}

func TestStringNetDevWithTypeUser(t *testing.T) {
	// The 'downscript' and 'script' parameters are not valid for 'user'
	// device type so we don't render them to the string in that case even
	// if they are populated in the type.
	testHostPorts := []portfwd{{proto: "tcp", port: 80}, {proto: "tcp", port: 443}}
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

func checkQemuString(qr Stringer, expected string, t *testing.T) {
	actual := qr.String()
	if expected != actual {
		t.Errorf("Rendered string %q not %q", actual, expected)
	}
}

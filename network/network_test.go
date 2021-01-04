package network_test

import (
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/nanovms/ops/network"
)

func TestGetNetworkInterfaces(t *testing.T) {
	ifcs, err := net.Interfaces()
	if err != nil {
		t.Error(err)
	}

	for _, ifc := range ifcs {
		if strings.HasPrefix(ifc.Name, "br") {
			// fmt.Printf("%+v\n", ifc)
		}
	}
}

func TestCreateAndDeleteBridge(t *testing.T) {

	brExists, err := network.CheckNetworkInterfaceExists("br1-test")
	if err != nil {
		t.Error(err)
	}

	if brExists == true {
		t.Errorf("br1-test should not exist")
	}

	_, err = network.AddBridge("br1-test")
	if err != nil {
		t.Errorf("%+v\n", err)
	}

	brExists, err = network.CheckNetworkInterfaceExists("br1-test")
	if err != nil {
		t.Error(err)
	}

	if brExists == false {
		t.Errorf("bridge was created in this test and should exist")
	}

	_, err = network.DeleteNIC("br1-test")
	if err != nil {
		t.Errorf("%+v\n", err)
	}

	brExists, err = network.CheckNetworkInterfaceExists("br1-test")
	if err != nil {
		t.Error(err)
	}

	if brExists == true {
		t.Errorf("br1-test should not exist after deleting")
	}
}

func TestCreateAndDeleteTAP(t *testing.T) {

	brExists, err := network.CheckNetworkInterfaceExists("tap1-test")
	if err != nil {
		t.Error(err)
	}

	if brExists == true {
		t.Errorf("tap1-test should not exist")
	}

	_, err = network.AddTap("tap1-test")
	if err != nil {
		t.Errorf("%+v\n", err)
	}

	brExists, err = network.CheckNetworkInterfaceExists("tap1-test")
	if err != nil {
		t.Error(err)
	}

	if brExists == false {
		t.Errorf("bridge was created in this test and should exist")
	}

	_, err = network.DeleteNIC("tap1-test")
	if err != nil {
		t.Errorf("%+v\n", err)
	}

	brExists, err = network.CheckNetworkInterfaceExists("tap1-test")
	if err != nil {
		t.Error(err)
	}

	if brExists == true {
		t.Errorf("tap1-test should not exist after deleting")
	}
}

func TestAddTapToBridge(t *testing.T) {
	_, err := network.AddBridge("br0-test")
	if err != nil {
		t.Error(err)
	}

	_, err = network.AddTap("tap0-test")
	if err != nil {
		t.Error(err)
	}

	check, err := network.CheckBridgeHasInterface("br0-test", "tap0-test")
	if err != nil {
		t.Error(err)
	}

	if check == true {
		t.Error("br0-test should not have tap0-test")
	}

	_, err = network.AddTapToBridge("br0-test", "tap0-test")
	if err != nil {
		t.Error(err)
	}

	check, err = network.CheckBridgeHasInterface("br0-test", "tap0-test")
	if err != nil {
		t.Error(err)
	}

	if check == false {
		t.Error("br0-test should have tap0-test")
	}

	bridgeIfcs, err := network.GetBridgeInterfacesNames("br0-test")
	if err != nil {
		t.Error(err)
	}

	bridgeIfcsWant := []string{"tap0-test"}

	if !reflect.DeepEqual(bridgeIfcs, bridgeIfcsWant) {
		t.Errorf("got %v, want %v", bridgeIfcs, bridgeIfcsWant)
	}

	_, err = network.DeleteNIC("tap0-test")
	if err != nil {
		t.Error(err)
	}

	_, err = network.DeleteNIC("br0-test")
	if err != nil {
		t.Error(err)
	}
}

func TestTurnNetworkInterfaceUp(t *testing.T) {
	_, err := network.AddTap("tap0-test")
	if err != nil {
		t.Error(err)
	}

	check, err := network.IsNIUp("tap0-test")
	if err != nil {
		t.Error(err)
	}

	if check == true {
		t.Error("tap0-test should not be up")
	}

	_, err = network.TurnNIUp("tap0-test")
	if err != nil {
		t.Error(err)
	}

	check, err = network.IsNIUp("tap0-test")
	if err != nil {
		t.Error(err)
	}

	if check == false {
		t.Error("tap0-test should be up")
	}

	_, err = network.TurnNIDown("tap0-test")
	if err != nil {
		t.Error(err)
	}

	check, err = network.IsNIUp("tap0-test")
	if err != nil {
		t.Error(err)
	}

	if check == true {
		t.Error("tap0-test should not be up")
	}

	_, err = network.DeleteNIC("tap0-test")
	if err != nil {
		t.Error(err)
	}

}

func TestSetIP(t *testing.T) {
	_, err := network.AddTap("tap0-test")
	if err != nil {
		t.Error(err)
	}

	_, err = network.SetNIIP("tap0-test", "192.168.1.66", "255.255.255.0")
	if err != nil {
		t.Error(err)
	}

	address, err := network.GetNetworkInterfaceIP("tap0-test")
	if err != nil {
		t.Error(err)
	}

	if address != "192.168.1.66" {
		t.Errorf("expected %s, got %s", "192.168.1.66", address)
	}

	_, err = network.DeleteNIC("tap0-test")
	if err != nil {
		t.Error(err)
	}
}

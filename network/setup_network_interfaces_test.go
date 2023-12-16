package network_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nanovms/ops/network"
	mock_network "github.com/nanovms/ops/network/mocks"
)

func TestSetupNetworkInterfaces(t *testing.T) {

	t.Run("should create tap if it does not exist and turn it on", func(t *testing.T) {
		networkService := NewNetworkService(t)
		tapDeviceName := "tap0-test"

		networkService.
			EXPECT().
			CheckNetworkInterfaceExists(tapDeviceName).
			Return(false, nil)

		networkService.
			EXPECT().
			AddTap(tapDeviceName).
			Return("success", nil)

		networkService.
			EXPECT().
			IsNIUp(tapDeviceName).
			Return(false, nil)

		networkService.
			EXPECT().
			TurnNIUp(tapDeviceName).
			Return("success", nil)

		network.SetupNetworkInterfaces(networkService, tapDeviceName, "", "", "", "")
	})

	t.Run("should not create tap or turn it on if it already exists and is active", func(t *testing.T) {
		networkService := NewNetworkService(t)
		tapDeviceName := "tap0-test"

		networkService.
			EXPECT().
			CheckNetworkInterfaceExists(tapDeviceName).
			Return(true, nil)

		networkService.
			EXPECT().
			IsNIUp(tapDeviceName).
			Return(true, nil)

		network.SetupNetworkInterfaces(networkService, tapDeviceName, "", "", "", "")
	})

	t.Run("should turn the tap up if it exists and is not active", func(t *testing.T) {
		networkService := NewNetworkService(t)
		tapDeviceName := "tap0-test"

		networkService.
			EXPECT().
			CheckNetworkInterfaceExists(tapDeviceName).
			Return(true, nil)

		networkService.
			EXPECT().
			IsNIUp(tapDeviceName).
			Return(false, nil)

		networkService.
			EXPECT().
			TurnNIUp(tapDeviceName).
			Return("success", nil)

		network.SetupNetworkInterfaces(networkService, tapDeviceName, "", "", "", "")
	})

	t.Run("should create bridge and turn it up if it does not exist and add tap to bridge network", func(t *testing.T) {
		networkService := NewNetworkService(t)
		tapDeviceName := "tap0-test"
		bridgeName := "br1"

		networkService.
			EXPECT().
			CheckNetworkInterfaceExists(tapDeviceName).
			Return(true, nil)

		networkService.
			EXPECT().
			CheckNetworkInterfaceExists(bridgeName).
			Return(false, nil)

		networkService.
			EXPECT().
			AddBridge(bridgeName).
			Return("success", nil)

		networkService.
			EXPECT().
			CheckBridgeHasInterface(bridgeName, tapDeviceName).
			Return(false, nil)

		networkService.
			EXPECT().
			AddTapToBridge(bridgeName, tapDeviceName).
			Return("success", nil)

		networkService.
			EXPECT().
			IsNIUp(bridgeName).
			Return(false, nil)

		networkService.
			EXPECT().
			TurnNIUp(bridgeName).
			Return("success", nil)

		networkService.
			EXPECT().
			IsNIUp(tapDeviceName).
			Return(true, nil)

		network.SetupNetworkInterfaces(networkService, tapDeviceName, bridgeName, "", "", "")
	})

	t.Run("should set IP in bridge if it already exists", func(t *testing.T) {
		networkService := NewNetworkService(t)
		tapDeviceName := "tap0-test"
		bridgeName := "br1"

		networkService.
			EXPECT().
			CheckNetworkInterfaceExists(tapDeviceName).
			Return(true, nil)

		networkService.
			EXPECT().
			CheckNetworkInterfaceExists(bridgeName).
			Return(true, nil)

		networkService.
			EXPECT().
			CheckBridgeHasInterface(bridgeName, tapDeviceName).
			Return(true, nil)

		networkService.
			EXPECT().
			IsNIUp(bridgeName).
			Return(true, nil)

		networkService.
			EXPECT().
			GetNetworkInterfaceIP(bridgeName).
			Return("", nil)

		networkService.
			EXPECT().
			FlushIPFromNI(bridgeName).
			Return("success", nil)

		networkService.
			EXPECT().
			SetNIIP(bridgeName, "192.168.1.66", "255.255.255.0").
			Return("success", nil)

		networkService.
			EXPECT().
			IsNIUp(tapDeviceName).
			Return(true, nil)

		network.SetupNetworkInterfaces(networkService, tapDeviceName, bridgeName, "192.168.1.100", "255.255.255.0", "")
	})

	t.Run("should not set IP in bridge if it is already set", func(t *testing.T) {
		networkService := NewNetworkService(t)
		tapDeviceName := "tap0-test"
		bridgeName := "br1"

		networkService.
			EXPECT().
			CheckNetworkInterfaceExists(tapDeviceName).
			Return(true, nil)

		networkService.
			EXPECT().
			CheckNetworkInterfaceExists(bridgeName).
			Return(true, nil)

		networkService.
			EXPECT().
			CheckBridgeHasInterface(bridgeName, tapDeviceName).
			Return(true, nil)

		networkService.
			EXPECT().
			IsNIUp(bridgeName).
			Return(true, nil)

		networkService.
			EXPECT().
			GetNetworkInterfaceIP(bridgeName).
			Return("192.168.1.66", nil)

		networkService.
			EXPECT().
			IsNIUp(tapDeviceName).
			Return(true, nil)

		network.SetupNetworkInterfaces(networkService, tapDeviceName, bridgeName, "192.168.1.100", "255.255.255.0", "")
	})

}

func NewNetworkService(t *testing.T) *mock_network.MockService {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	return mock_network.NewMockService(ctrl)
}

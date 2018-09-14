package main
import (
    "github.com/vishvananda/netlink"
    "net"
    "os/exec"
    "errors"
)

// TODO: Remove hard coding
const bridgeName string = "br0"
const tapName string = "tap0"

func createBridgeNetwork(adapterName string) (*netlink.Bridge, error) {
    // create a bridge named br0
    la := netlink.NewLinkAttrs()
    la.Name = bridgeName
    bridge := &netlink.Bridge{LinkAttrs: la}
    err := netlink.LinkAdd(bridge)
    if err != nil  {
        return nil, err
    }
    // bring up the bridge
    if err = netlink.LinkSetUp(bridge); err != nil {
        return nil, err
    }
    // add network adapter to bridge
    eth0 , err := netlink.LinkByName(adapterName)
    if err = netlink.LinkSetMaster(eth0, bridge); err != nil {
        return nil, err
    }
    // add tap0
    tap0 := &netlink.Tuntap {
        LinkAttrs: netlink.LinkAttrs{Name: tapName},
        Mode: netlink.TUNTAP_MODE_TAP,
    }
    err = netlink.LinkAdd(tap0)
    if err != nil {
        return nil, err
    }
    // bring up tap0
    if err = netlink.LinkSetUp(tap0); err != nil {
        return nil, err
    }
    if err = netlink.LinkSetMaster(tap0, bridge); err != nil {
        return nil, err
    }
    return bridge, nil
}

func findFirstActiveAdapter() (*net.Interface, error) {
    interfaces , err := net.Interfaces()
    if err != nil {
        return nil, err
    }
    var eth0 *net.Interface
    for _,eif := range interfaces{
        if eif.Flags & net.FlagUp > 0 {
            // ugly: Need a better way to pick wired ethernet adapter.
            if eif.Name[0] == 'e' {
                eth0 = &eif 
                break
            }
        }
    }
    if eth0 == nil {
        return nil, errors.New("No active ethernet adapter found")
    }
    return eth0, nil
}

// TODO : this is not portable.
func assignIP(bridge *netlink.Bridge) error {
    args := []string{"-v",bridge.Name}
    cmd := exec.Command("dhclient", args...)
    if err := cmd.Run(); err != nil {
        return err
    }
    return nil
}

func setupBridgeNetwork() error {
    var err error
    eth0, err := findFirstActiveAdapter()
    if err != nil {
        return err
    }
    bridge, err := createBridgeNetwork(eth0.Name)
    if err != nil {
        return err
    }
    err = assignIP(bridge)
    if err != nil {
        return err
    }
    return nil
}

func resetBridgeNetwork() error {
    eth0, err := findFirstActiveAdapter()
    if err != nil {
            return err
    }
    adapterName := eth0.Name
    br0 , err := netlink.LinkByName(bridgeName)
    if err != nil {
        return err
    } 
    adapter , err := netlink.LinkByName(adapterName)
    if err != nil {
        return err
    }
    if err = netlink.LinkSetNoMaster(adapter); err != nil {
        return err
    }
    tap0, err := netlink.LinkByName(tapName)
    if err = netlink.LinkSetNoMaster(tap0); err != nil {
        return err
    }
    if err = netlink.LinkSetDown(tap0); err != nil {
        return err
    }
    if err = netlink.LinkSetDown(br0); err != nil {
        return err
    }
    if err = netlink.LinkDel(tap0);err != nil {
        return err
    }
    if err = netlink.LinkDel(br0);err != nil{
        return err
    }
    return nil
}
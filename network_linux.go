package main

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

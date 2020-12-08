package lepton

import "testing"

func TestGetAzureVirtualNetworkFromID(t *testing.T) {
	got := getAzureVirtualNetworkFromID("/subscriptions/c3738b2f-eb5f-46a2-9090-267d3c522371/resourceGroups/fabio/providers/Microsoft.Network/virtualNetworks/walk-server-1607352001/subnets/walk-server-1607352001")
	want := "walk-server-1607352001"

	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

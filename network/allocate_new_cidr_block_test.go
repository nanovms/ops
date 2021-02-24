package network_test

import (
	"testing"

	"github.com/nanovms/ops/network"
	"gotest.tools/assert"
)

func TestAllocateNewCidrBlock(t *testing.T) {
	t.Run("should return empty string if no cidr block is associated", func(t *testing.T) {
		cidrBlocks := []string{}

		assert.Equal(t, network.AllocateNewCidrBlock(cidrBlocks), "")
	})

	t.Run("(netmask/16) should return a cidr block with an IP range higher than the associated cidr blocks", func(t *testing.T) {
		cidrBlocks := []string{"10.0.0.0/16", "172.31.0.0/16"}

		assert.Equal(t, network.AllocateNewCidrBlock(cidrBlocks), "172.32.0.0/16")
	})

	t.Run("(netmask/24) should return a cidr block with an IP range higher than the associated cidr blocks", func(t *testing.T) {
		cidrBlocks := []string{"10.0.0.0/24", "172.31.0.0/24"}

		assert.Equal(t, network.AllocateNewCidrBlock(cidrBlocks), "172.31.1.0/24")
	})

	t.Run("(netmask/16) should increment first 8 bit part", func(t *testing.T) {
		cidrBlocks := []string{"10.0.0.0/16", "172.255.0.0/16"}

		assert.Equal(t, network.AllocateNewCidrBlock(cidrBlocks), "173.0.0.0/16")
	})

	t.Run("(netmask/24) should increment second 8 bit part", func(t *testing.T) {
		cidrBlocks := []string{"10.0.0.0/16", "172.1.255.0/24"}

		assert.Equal(t, network.AllocateNewCidrBlock(cidrBlocks), "172.2.0.0/24")
	})

	t.Run("(netmask/24) should increment fist 8 bit part", func(t *testing.T) {
		cidrBlocks := []string{"10.0.0.0/16", "172.255.255.0/24"}

		assert.Equal(t, network.AllocateNewCidrBlock(cidrBlocks), "173.0.0.0/24")
	})
}

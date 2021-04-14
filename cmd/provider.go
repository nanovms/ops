package cmd

import (
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider"
	"github.com/nanovms/ops/types"
)

func getProviderAndContext(c *types.Config, providerName string) (api.Provider, *api.Context, error) {
	p, err := provider.CloudProvider(providerName, &c.CloudConfig)
	if err != nil {
		return nil, nil, err
	}

	ctx := api.NewContext(c)

	return p, ctx, nil
}

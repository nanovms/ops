//go:build digitalocean || do || !onlyprovider

package digitalocean

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/nanovms/ops/lepton"
)

// ListVpcs - List all VPCs
func (do *DigitalOcean) GetVPC(ctx *lepton.Context, zone, vpcName string) (*godo.VPC, error) {

	if vpcName == "" {
		return nil, nil
	} else if zone == "" && vpcName != "" {
		ctx.Logger().Debugf("zone is required to get vpc")
		return nil, fmt.Errorf("zone is required to get vpc")
	}

	page := 1
	var vpc *godo.VPC

	// ctx.Logger().Debug("getting all vpcs")

	for {
		opts := &godo.ListOptions{
			Page:    page,
			PerPage: 200, // max allowed by DO
		}

		vpcs, _, err := do.Client.VPCs.List(context.TODO(), opts)
		if err != nil {
			return nil, err
		}
		if len(vpcs) == 0 {
			break
		}

		for _, v := range vpcs {
			if v.Name == vpcName && v.RegionSlug == zone {
				if vpc != nil {
					ctx.Logger().Debugf("found another vpc %s that matches the criteria %s", v.ID, vpcName)
				}
				vpc = v
			}
		}
		page++
	}

	if vpc == nil {
		ctx.Logger().Debugf("no vpcs with name %s found", vpcName)
		return nil, fmt.Errorf("vpc %s not found", vpcName)
	}

	return vpc, nil
}

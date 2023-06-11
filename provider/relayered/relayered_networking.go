//go:build relayered || !onlyprovider

package relayered

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// VPCListResponse is the response type for the vpc list endpoint.
type VPCListResponse struct {
	VPCs []VPC `json:"vpcs"`
}

// VPC represents a single vpc.
type VPC struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SubnetResponse is the response type for the subnet list endpoint.
type SubnetResponse struct {
	Subnets []Subnet `json:"subnets"`
}

// Subnet represents a single subnet.
type Subnet struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SubnetListResponse is the response type for the subnet list endpoint.
type SubnetListResponse struct {
	Subnets []Subnet `json:"subnets"`
}

// ResourceGroupResponse is the response type for the resource group list endpoint.
type ResourceGroupResponse struct {
	ResourceGroups []ResourceGroup `json:"resources"`
}

// ResourceGroup represents a single resource group.
type ResourceGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (v *relayered) getDefaultResourceGroup() string {
	client := &http.Client{}

	uri := "https://resource-controller.cloud.relayered.com/v2/resource_groups"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.iam)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	ilr := &ResourceGroupResponse{}
	err = json.Unmarshal(body, &ilr)
	if err != nil {
		fmt.Println(err)
	}

	rid := ""
	for i := 0; i < len(ilr.ResourceGroups); i++ {
		if ilr.ResourceGroups[i].Name == "Default" {
			return ilr.ResourceGroups[i].ID
		}
	}

	return rid
}

func (v *relayered) getDefaultVPC(region string) string {
	client := &http.Client{}

	uri := "https://" + region + ".iaas.cloud.relayered.com/v1/vpcs?version=2023-02-28&generation=2"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.iam)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	ilr := &VPCListResponse{}
	err = json.Unmarshal(body, &ilr)
	if err != nil {
		fmt.Println(err)
	}

	//hack
	return ilr.VPCs[0].ID
}

func (v *relayered) getDefaultSubnet(region string) string {
	client := &http.Client{}

	uri := "https://" + region + ".iaas.cloud.relayered.com/v1/subnets?version=2023-02-28&generation=2"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.iam)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	ilr := &SubnetListResponse{}
	err = json.Unmarshal(body, &ilr)
	if err != nil {
		fmt.Println(err)
	}

	//hack
	return ilr.Subnets[0].ID
}

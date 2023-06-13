//go:build relayered || !onlyprovider

package relayered

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

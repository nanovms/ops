//go:build upcloud || !onlyprovider

//go:generate mockgen -source=$GOFILE -destination=$PWD/mocks/${GOFILE} -package=mocks

package upcloud

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud/client"
	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud/service"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "upcloud"

// Service is the interface implemented by upcloud service
type Service interface {
	service.Server
	service.Storage
	service.Tag
	service.Account
}

// Provider provides access to the UpCloud API.
type Provider struct {
	upcloud Service
	zone    string
}

// NewProvider UpCloud
func NewProvider() *Provider {
	return &Provider{}
}

// NewProviderWithService returns an instance of Upcloud provider and initializes upcloud service
func NewProviderWithService(service Service) *Provider {
	return &Provider{upcloud: service}
}

// Initialize checks conditions to use upcloud
func (p *Provider) Initialize(c *types.ProviderConfig) error {
	user := os.Getenv("UPCLOUD_USER")
	if user == "" {
		return errors.New(`"UPCLOUD_USER" not set`)
	}

	password := os.Getenv("UPCLOUD_PASSWORD")
	if password == "" {
		return errors.New(`"UPCLOUD_PASSWORD" not set`)
	}

	p.zone = c.Zone
	if p.zone == "" {
		p.zone = os.Getenv("UPCLOUD_ZONE")
		if p.zone == "" {
			return errors.New(`"UPCLOUD_ZONE" not set`)
		}
	}

	if p.upcloud == nil {
		c := client.New(user, password, client.WithTimeout(time.Second*600))
		p.upcloud = service.New(c)
	}

	_, err := p.upcloud.GetAccount(context.Background())

	if err != nil {
		if serviceError, ok := err.(*upcloud.Problem); ok {
			return errors.New(serviceError.Title)
		}
		return errors.New("Invalid credentials")
	}

	return nil
}

func (p *Provider) findOrCreateTag(tag upcloud.Tag) (upcloudTag *upcloud.Tag, err error) {
	tagsResponse, err := p.upcloud.GetTags(context.Background())
	if err != nil {
		return
	}

	for _, t := range tagsResponse.Tags {
		if t.Name == tag.Name {
			upcloudTag = &t
			return
		}
	}

	createTagReq := &request.CreateTagRequest{Tag: tag}

	upcloudTag, err = p.upcloud.CreateTag(context.Background(), createTagReq)
	if err != nil {
		return
	}

	return
}

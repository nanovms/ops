package gcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nanovms/ops/constants"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	dns "google.golang.org/api/dns/v1"
)

// ProviderName of the cloud platform provider
const ProviderName = "gcp"

var (
	errGCloudProjectIDMissing = func() error { return errors.New("projectid is missing. Please set env variable GCLOUD_PROJECT_ID") }
	errGCloudZoneMissing      = func() error { return errors.New("zone is missing. Please set env variable GCLOUD_ZONE") }
)

// GCloudOperation status check
type GCloudOperation struct {
	service       *compute.Service
	projectID     string
	name          string
	area          string
	operationType string
}

func buildGcpTags(tags []types.Tag) map[string]string {
	labels := map[string]string{}
	for _, tag := range tags {
		labels[tag.Key] = tag.Value
	}

	labels["createdby"] = "ops"

	return labels
}

func checkGCCredentialsProvided() error {
	creds, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
	if !ok {
		return fmt.Errorf(constants.ErrorColor, "error: GOOGLE_APPLICATION_CREDENTIALS not set.\nFollow https://cloud.google.com/storage/docs/reference/libraries to set it up.\n")
	}
	if _, err := os.Stat(creds); os.IsNotExist(err) {
		return fmt.Errorf(constants.ErrorColor, fmt.Sprintf("error: File %s mentioned in GOOGLE_APPLICATION_CREDENTIALS does not exist.", creds))
	}
	return nil
}

func (gop *GCloudOperation) isDone(ctx context.Context) (bool, error) {
	var (
		op  *compute.Operation
		err error
	)
	fmt.Printf(".")
	switch gop.operationType {
	case "zone":
		op, err = gop.service.ZoneOperations.Get(gop.projectID, gop.area, gop.name).Context(ctx).Do()
	case "region":
		op, err = gop.service.RegionOperations.Get(gop.projectID, gop.area, gop.name).Context(ctx).Do()
	case "global":
		op, err = gop.service.GlobalOperations.Get(gop.projectID, gop.name).Context(ctx).Do()
	default:
		fmt.Printf("Unexpected error happened. Unknown operation type: %s", gop.operationType)
		os.Exit(1)
	}
	if err != nil {
		return false, err
	}
	if op == nil || op.Status != "DONE" {
		return false, nil
	}
	if op.Error != nil && len(op.Error.Errors) > 0 && op.Error.Errors[0] != nil {
		e := op.Error.Errors[0]
		return false, fmt.Errorf("%v - %v", e.Code, e.Message)
	}

	return true, nil
}

// GCloud Provider to interact with GCP cloud infrastructure
type GCloud struct {
	Storage    *Storage
	Service    *compute.Service
	ProjectID  string
	Zone       string
	dnsService *dns.Service
}

// NewProvider GCP
func NewProvider() *GCloud {
	return &GCloud{}
}

func (p *GCloud) pollOperation(ctx context.Context, projectID string, service *compute.Service, op compute.Operation) error {
	var area, operationType string

	if strings.Contains(op.SelfLink, "zone") {
		s := strings.Split(op.Zone, "/")
		operationType = "zone"
		area = s[len(s)-1]
	} else if strings.Contains(op.SelfLink, "region") {
		s := strings.Split(op.Region, "/")
		operationType = "region"
		area = s[len(s)-1]
	} else {
		operationType = "global"
	}

	gOp := &GCloudOperation{
		service:       service,
		projectID:     projectID,
		name:          op.Name,
		area:          area,
		operationType: operationType,
	}

	var pollCount int
	for {
		pollCount++

		status, err := gOp.isDone(ctx)
		if err != nil {
			fmt.Printf("Operation %s failed.\n", op.Name)
			return err
		}
		if status {
			break
		}
		// Wait for 120 seconds
		if pollCount > 60 {
			return fmt.Errorf("\nOperation timed out. No of tries %d", pollCount)
		}
		// TODO: Rate limit API instead of time.Sleep
		time.Sleep(2 * time.Second)
	}
	fmt.Printf("\nOperation %s completed successfully.\n", op.Name)
	return nil
}

// Initialize GCP related things
func (p *GCloud) Initialize(config *types.ProviderConfig) error {
	p.Storage = &Storage{}

	if config.ProjectID == "" {
		return fmt.Errorf("ProjectID missing")
	}

	if config.Zone == "" {
		return fmt.Errorf("Zone missing")
	}

	if err := checkGCCredentialsProvided(); err != nil {
		return err
	}

	client, err := google.DefaultClient(context.Background(), compute.CloudPlatformScope)
	if err != nil {
		return err
	}

	computeService, err := compute.New(client)
	if err != nil {
		return err
	}

	p.Service = computeService

	p.dnsService, err = p.getDNSService()
	if err != nil {
		return err
	}

	return nil
}

// GetStorage returns storage interface for cloud provider
func (p *GCloud) GetStorage() lepton.Storage {
	return p.Storage
}

package lepton

import (
	"fmt"
	"strings"
)

var (
	// ErrInstanceNotFound is used when an instance doesn't exist in provider
	ErrInstanceNotFound = func(id string) error { return fmt.Errorf("Instance with id %v not found", id) }
)

var (
	// TTLDefault is the default ttl value used to create DNS records
	TTLDefault = 300
)

// Provider is an interface that provider must implement
type Provider interface {
	Initialize() error

	BuildImage(ctx *Context) (string, error)
	BuildImageWithPackage(ctx *Context, pkgpath string) (string, error)
	CreateImage(ctx *Context) error
	ListImages(ctx *Context) error
	GetImages(ctx *Context) ([]CloudImage, error)
	DeleteImage(ctx *Context, imagename string) error
	ResizeImage(ctx *Context, imagename string, hbytes string) error
	SyncImage(config *Config, target Provider, imagename string) error
	customizeImage(ctx *Context) (string, error)

	CreateInstance(ctx *Context) error
	ListInstances(ctx *Context) error
	GetInstances(ctx *Context) ([]CloudInstance, error)
	GetInstanceByID(ctx *Context, id string) (*CloudInstance, error)
	DeleteInstance(ctx *Context, instancename string) error
	StopInstance(ctx *Context, instancename string) error
	StartInstance(ctx *Context, instancename string) error
	GetInstanceLogs(ctx *Context, instancename string) (string, error)
	PrintInstanceLogs(ctx *Context, instancename string, watch bool) error

	VolumeService

	GetStorage() Storage
}

// Storage is an interface that provider's storage must implement
type Storage interface {
	CopyToBucket(config *Config, source string) error
}

// VolumeService is an interface for volume related operations
type VolumeService interface {
	CreateVolume(config *Config, name, data, size, provider string) (NanosVolume, error)
	GetAllVolumes(config *Config) error
	DeleteVolume(config *Config, name string) error
	AttachVolume(config *Config, image, name, mount string) error
	DetachVolume(config *Config, image, name string) error
}

// DNSRecord is ops representation of a dns record
type DNSRecord struct {
	Name string
	IP   string
	Type string
	TTL  int
}

// DNSService is an interface for DNS related operations
type DNSService interface {
	FindOrCreateZoneIDByName(config *Config, name string) (string, error)
	DeleteZoneRecordIfExists(config *Config, zoneID string, recordName string) error
	CreateZoneRecord(config *Config, zoneID string, record *DNSRecord) error
}

// CreateDNSRecord does the necessary operations to create a DNS record without issues in an cloud provider
func CreateDNSRecord(config *Config, aRecordIP string, dnsService DNSService) error {
	domainName := config.RunConfig.DomainName
	if err := isDomainValid(domainName); err != nil {
		return err
	}

	domainParts := strings.Split(domainName, ".")

	// example:
	// domainParts := []string{"test","example","com"}
	zoneName := domainParts[len(domainParts)-2]                 // example
	dnsName := zoneName + "." + domainParts[len(domainParts)-1] // example.com
	aRecordName := domainName + "."                             // test.example.com

	zoneID, err := dnsService.FindOrCreateZoneIDByName(config, dnsName)
	if err != nil {
		return err
	}

	err = dnsService.DeleteZoneRecordIfExists(config, zoneID, aRecordName)
	if err != nil {
		return err
	}

	record := &DNSRecord{
		Name: aRecordName,
		IP:   aRecordIP,
		Type: "A",
		TTL:  TTLDefault,
	}
	err = dnsService.CreateZoneRecord(config, zoneID, record)
	if err != nil {
		return err
	}

	return nil
}

// Context captures required info for provider operation
type Context struct {
	config   *Config
	provider *Provider
}

// NewContext Create a new context for the given provider
// valid providers are "gcp", "aws" and "onprem"
func NewContext(c *Config, provider *Provider) *Context {
	return &Context{
		config:   c,
		provider: provider,
	}
}

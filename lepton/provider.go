package lepton

import (
	"fmt"
	"os"
	"strings"

	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

var (
	// TTLDefault is the default ttl value used to create DNS records
	TTLDefault = 300

	// Prefix message for instance not found error
	errPrefixInstanceNotFound = "instance not found"
)

// Provider is an interface that provider must implement
type Provider interface {
	Initialize(config *types.ProviderConfig) error

	BuildImage(ctx *Context) (string, error)
	BuildImageWithPackage(ctx *Context, pkgpath string) (string, error)
	CreateImage(ctx *Context, imagePath string) error
	ListImages(ctx *Context) error
	GetImages(ctx *Context) ([]CloudImage, error)
	DeleteImage(ctx *Context, imagename string) error
	ResizeImage(ctx *Context, imagename string, hbytes string) error
	SyncImage(config *types.Config, target Provider, imagename string) error
	CustomizeImage(ctx *Context) (string, error)

	CreateInstance(ctx *Context) error
	ListInstances(ctx *Context) error
	GetInstances(ctx *Context) ([]CloudInstance, error)
	GetInstanceByName(ctx *Context, name string) (*CloudInstance, error)
	DeleteInstance(ctx *Context, instancename string) error
	StopInstance(ctx *Context, instancename string) error
	StartInstance(ctx *Context, instancename string) error
	RebootInstance(ctx *Context, instancename string) error
	GetInstanceLogs(ctx *Context, instancename string) (string, error)
	PrintInstanceLogs(ctx *Context, instancename string, watch bool) error

	VolumeService
}

// Mirrorer is an interface that all provider which provider feature to
// copy an image from one region to another
type Mirrorer interface {
	MirrorImage(ctx *Context, imageName, srcRegion, dstRegion string) (string, error)
}

// ErrInstanceNotFound creates new error stating instance with given name cannot be found
func ErrInstanceNotFound(name string) error {
	return fmt.Errorf("%s: %s", errPrefixInstanceNotFound, name)
}

// IsInstanceNotFoundError checks if given error is an ErrInstanceNotFound
func IsInstanceNotFoundError(err error) bool {
	return strings.HasPrefix(err.Error(), errPrefixInstanceNotFound)
}

// Storage is an interface that provider's storage must implement
type Storage interface {
	CopyToBucket(config *types.Config, source string) error
}

// VolumeService is an interface for volume related operations
type VolumeService interface {
	CreateVolumeImage(ctx *Context, imageName, data, provider string) (NanosVolume, error)
	CreateVolumeFromSource(ctx *Context, sourceType, sourceName, name, provider string) error

	CreateVolume(ctx *Context, volumeName, data, provider string) (NanosVolume, error)
	GetAllVolumes(ctx *Context) (*[]NanosVolume, error)
	DeleteVolume(ctx *Context, volumeName string) error
	AttachVolume(ctx *Context, instanceName, volumeName string, attachID int) error
	DetachVolume(ctx *Context, instanceName, volumeName string) error
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
	FindOrCreateZoneIDByName(config *types.Config, name string) (string, error)
	DeleteZoneRecordIfExists(config *types.Config, zoneID string, recordName string) error
	CreateZoneRecord(config *types.Config, zoneID string, record *DNSRecord) error
}

// CreateDNSRecord does the necessary operations to create a DNS record without issues in an cloud provider
func CreateDNSRecord(config *types.Config, aRecordIP string, dnsService DNSService) error {
	domainName := config.CloudConfig.DomainName
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
	config *types.Config
	logger *log.Logger
}

// Config returns context configuration
func (c Context) Config() *types.Config {
	return c.config
}

// Logger returns logger
func (c Context) Logger() *log.Logger {
	return c.logger
}

// NewContext Create a new context for the given provider
// valid providers are "gcp", "aws" and "onprem"
func NewContext(c *types.Config) *Context {

	logger := log.New(os.Stdout)

	if c.RunConfig.ShowDebug {
		logger.SetDebug(true)
		logger.SetError(true)
		logger.SetWarn(true)
		logger.SetInfo(true)
	}

	if c.RunConfig.ShowWarnings {
		logger.SetWarn(true)
	}

	if c.RunConfig.ShowErrors {
		logger.SetError(true)
	}

	if c.RunConfig.Verbose {
		logger.SetInfo(true)
	}

	return &Context{
		config: c,
		logger: logger,
	}
}

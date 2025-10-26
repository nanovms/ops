package types

import (
	"encoding/json"
	"reflect"
)

// Config for Build
type Config struct {
	// Args defines an array of commands to execute when the image is launched.
	Args []string `json:",omitempty"`

	// Disable auto copy of files from host to container when present in args
	DisableArgsCopy bool `json:",omitempty"`

	// BaseVolumeSz is an optional parameter for defining the size of the base
	// volume (defaults to the end of blocks written by TFS).
	BaseVolumeSz string `json:",omitempty"`

	// Boot
	Boot string `json:",omitempty"`

	// Boot path of UEFI bootloader file
	UefiBoot string `json:",omitempty"`

	// Uefi indicates whether image should support booting via UEFI
	Uefi bool `json:",omitempty"`

	// BuildDir
	BuildDir string `json:",omitempty"`

	// CloudConfig configures various attributes about the cloud provider.
	CloudConfig ProviderConfig `json:",omitempty"`

	// TargetConfig allows config that is pertinent to a specific
	// provider.
	TargetConfig map[string]string `json:",omitempty"`

	// Debugflags
	Debugflags []string `json:",omitempty"`

	// Dirs defines an array of directory locations to include into the image.
	Dirs []string `json:",omitempty"`

	// Env defines a map of environment variables to specify for the image
	// runtime.
	Env map[string]string `json:",omitempty"`

	// Files defines an array of file locations to include into the image.
	Files []string `json:",omitempty"`

	// Force
	Force bool `json:",omitempty"`

	// Home specifies the root folder for an ops home. By default it is
	// an empty string and not used. Any non-empty string value will
	// overrride anything that might be present in OPS_HOME env var.
	// This allows the user to utilize multiple OPS_HOME values for
	// different contexts in the same instantiation.
	Home string `json:"home,omitempty"`

	// Kernel
	Kernel string `json:",omitempty"`

	// Klibs host location
	KlibDir string `json:",omitempty"`

	// Klibs
	Klibs []string `json:",omitempty"`

	// MapDirs specifies a map of local directories to add to into the image.
	// These directory paths are then adjusted from local path specification
	// to image path specification.
	MapDirs map[string]string `json:",omitempty"`

	// Mounts
	Mounts map[string]string `json:",omitempty"`

	// NameServers is an optional parameter array that defines the DNS server to use
	// for DNS resolutions (defaults to Google's DNS server: '8.8.8.8').
	NameServers []string `json:",omitempty"`

	// NanosVersion
	NanosVersion string `json:",omitempty"`

	// NightlyBuild
	NightlyBuild bool `json:",omitempty"`

	// NoTrace
	NoTrace []string `json:",omitempty"`

	// Straight passthrough of options to manifest
	ManifestPassthrough map[string]interface{} `json:",omitempty"`

	// Program
	Program string `json:",omitempty"`

	// ProgramPath specifies the original path of the program to refer to on
	// attach/detach.
	ProgramPath string `json:",omitempty"`

	// RebootOnExit defines whether the image should automatically reboot
	// if an error/failure occurs.
	RebootOnExit bool `json:",omitempty"`

	// RunConfig
	RunConfig RunConfig `json:",omitempty"`

	// LocalFilesParentDirectory is the parent directory of the files/directories specified in Files and Dirs
	// The default value is the directory from where the ops command is running
	LocalFilesParentDirectory string `json:",omitempty"`

	// TargetRoot
	TargetRoot string `json:",omitempty"`

	// TFSv4 forces use of the deprecated TFS version 4 encoding
	TFSv4 bool `json:",omitempty"`

	// Version
	Version string `json:",omitempty"`

	// Language
	Language string `json:",omitempty"`

	// Description
	Description string `json:",omitempty"`

	// VolumesDir is the directory used to store and fetch volumes
	VolumesDir string `json:",omitempty"`

	// PackageBaseURL gives URL for downloading of packages
	PackageBaseURL string `json:",omitempty"`

	// PackageManifestURL stores info about all packages
	PackageManifestURL string `json:",omitempty"`
}

// ProviderConfig give provider details
type ProviderConfig struct {

	// BucketName specifies the bucket to store the ops built image artifacts.
	BucketName string `cloud:"bucketname" json:",omitempty"`

	// BucketNamespace is required on uploading files to cloud providers as oci
	BucketNamespace string `json:",omitempty"`

	// Enable confidential computing
	ConfidentialVM bool `json:",omitempty"`

	// DedicatedHostID
	DedicatedHostID string `json:",omitempty"`

	// DomainName
	DomainName string `json:",omitempty"`

	// Used by cloud provider to assign a public static IP to a NIC
	StaticIP string `json:",omitempty"`

	// EnableIPv6 enables IPv6 when creating a vpc. It does not affect an existing VPC
	EnableIPv6 bool `json:",omitempty"`

	// Flavor
	Flavor string `cloud:"flavor" json:",omitempty"`

	// ImageType
	ImageType string `cloud:"imagetype" json:",omitempty"`

	// ImageName
	ImageName string `cloud:"imagename" json:",omitempty"`

	// InstanceProfile is a container for an IAM role
	// you can use to pass role information to an EC2 instance when the instance starts.
	InstanceProfile string `json:",omitempty"`

	// KMS optionally encrypts AMIs if set. 'default' may be used for
	// the default key or a KMS arn may be specified.
	KMS string `json:",omitempty"`

	// Platform defines the cloud provider to use with the ops CLI, currently
	// supporting aws, azure, and gcp.
	Platform string `cloud:"platform" json:",omitempty"`

	// ProjectID is used to define the project ID when the Platform is set
	// to gcp.
	ProjectID string `cloud:"projectid" json:",omitempty"`

	// RootVolume are specific settings for the root volume.
	RootVolume CloudVolume `cloud:"root_volume" json:",omitempty"`

	// SecurityGroup
	SecurityGroup string `json:",omitempty"`

	// SkipImportVerify skips verifying that a vm importer role exists
	// on AWS for volume imports. The role might exist but the end-user
	// might not have permissions to verify that.
	SkipImportVerify bool `json:",omitempty"`

	// Spot enables spot provisioning
	Spot bool `json:",omitempty"`

	// Subnet
	Subnet string `json:",omitempty"`

	// Tags
	Tags []Tag `json:",omitempty"`

	// UserData contains cloud-init script or user data to be passed to the instance
	UserData string `json:",omitempty"`

	// VPC
	VPC string `json:",omitempty"`

	// Zone is used to define the location of the host resource. Lists of these
	// zones are dependent on selected Platform and can be found here:
	// aws: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html
	// azure: https://azure.microsoft.com/en-us/global-infrastructure/geographies/#overview
	// gcp: https://cloud.google.com/compute/docs/regions-zones#available
	Zone string `cloud:"zone" json:",omitempty"`
}

// Tag is used as property on creating instances
type Tag struct {
	// Key
	Key string `json:"key"`

	// Value
	Value string `json:"value"`

	// Attribute extended
	Attribute *TagAttribute `json:"attribute,omitempty"`
}

// TagAttribute ...
type TagAttribute struct {
	// Image Label Tag
	ImageLabel *bool `json:"image_label,omitempty"`

	// Instance Label Tag
	InstanceLabel *bool `json:"instance_label,omitempty"`

	// Instance Network Tag - will use the Value
	InstanceNetwork *bool `json:"instance_network,omitempty"`

	// Instance Metadata Tag
	InstanceMetadata *bool `json:"instance_metadata,omitempty"`
}

// HasAttribute checks if extended attributes are being used
func (t Tag) HasAttribute() bool {
	return !(t.Attribute == nil || reflect.ValueOf(*t.Attribute).IsZero())
}

// IsImageLabel ...
func (t Tag) IsImageLabel() bool {
	if !t.HasAttribute() {
		return true // backward compatibility
	}
	return t.Attribute.ImageLabel != nil && *t.Attribute.ImageLabel
}

// IsInstanceLabel ...
func (t Tag) IsInstanceLabel() bool {
	if !t.HasAttribute() {
		return true // backward compatibility
	}
	return t.Attribute.InstanceLabel != nil && *t.Attribute.InstanceLabel
}

// IsInstanceNetwork ...
func (t Tag) IsInstanceNetwork() bool {
	if !t.HasAttribute() {
		return false
	}
	return t.Attribute.InstanceNetwork != nil && *t.Attribute.InstanceNetwork
}

// IsInstanceMetadata ...
func (t Tag) IsInstanceMetadata() bool {
	if !t.HasAttribute() {
		return false
	}
	return t.Attribute.InstanceMetadata != nil && *t.Attribute.InstanceMetadata
}

// RunConfig provides runtime details
type RunConfig struct {
	// Accel defines whether hardware acceleration should be enabled.
	Accel bool `json:",omitempty"`

	// AtExit allows hooks to be ran after instance stops.
	AtExit string `json:",omitempty"`

	// Bridged parameter is set to true if bridged networking mode is
	// in use. This also enables KVM acceleration.
	Bridged bool `json:",omitempty"`

	// BridgeIPAddress is an optional ip address for a bridge when used
	// w/ops run.
	BridgeIPAddress string `json:",omitempty"`

	// BridgeName
	BridgeName string `json:",omitempty"`

	// CanIPForward enable IP forwarding when creating an instance on GCP
	CanIPForward bool `json:",omitempty"`

	// CPUs specifies the number of CPU cores to use
	CPUs int `json:",omitempty"`

	GPUs    int    `json:",omitempty"`
	GPUType string `json:",omitempty"`

	// Debug
	Debug bool `json:",omitempty"`

	// Gateway
	Gateway string `json:",omitempty"`

	// GdbPort
	GdbPort int `json:",omitempty"`

	// ImageName
	ImageName string `json:",omitempty"`

	// InstanceGroup
	InstanceGroup string `json:",omitempty"`

	// InstanceName
	InstanceName string `json:",omitempty"`

	// IPAddress
	IPAddress string `json:",omitempty"`

	// IPv6Address
	IPv6Address string `json:",omitempty"`

	// Kernel
	Kernel string `json:",omitempty"`

	// Memory configures the amount of memory to allocate to qemu (default
	// is 128 MiB). Optionally, a suffix of "M" or "G" can be used to
	// signify a value in megabytes or gigabytes respectively.
	Memory string `json:",omitempty"`

	// Mgmt is an optional mgmt port for onprem QMP access.
	Mgmt string `json:",omitempty"`

	// Vga whether to emulate a VGA output device
	Vga bool `json:",omitempty"`

	// host directories shared with guest via VirtFS
	VirtfsShares map[string]string `json:"-"`

	// Mounts
	Mounts []string `json:",omitempty"`

	// AttachVolumeOnInstanceCreate tries to attach the volumes configured in Mounts upon instance creation
	AttachVolumeOnInstanceCreate bool `json:",omitempty"`

	// NetMask
	NetMask string `json:",omitempty"`

	// Nics is a list of pre-configured network cards
	// Meant to eventually deprecate the existing single-nic configuration
	// Currently only supported for Proxmox
	Nics []Nic `json:",omitempty"`

	// Background runs unikernel in background
	// use onprem instances commands to manage the unikernel
	Background bool `json:",omitempty"`

	// Ports specifies a list of port to expose.
	Ports []string `json:",omitempty"`

	// QMP optionally turns on a QMP interface for the onprem target.
	QMP bool `json:",omitempty"`

	// ShowDebug
	ShowDebug bool `json:",omitempty"`

	// ShowErrors
	ShowErrors bool `json:",omitempty"`

	// ShowWarnings
	ShowWarnings bool `json:",omitempty"`

	// JSON output
	JSON bool `json:",omitempty"`

	// TapName
	TapName string `json:",omitempty"`

	// UDPPorts
	UDPPorts []string `json:",omitempty"`

	// Verbose enables logging for the runtime environment.
	Verbose bool `json:",omitempty"`

	// VolumeSizeInGb is an optional parameter only available for OpenStack.
	VolumeSizeInGb int `json:",omitempty"`

	// ThreadsPerCore: The number of threads per physical core. To disable
	// simultaneous multithreading (SMT) set this to 1. If unset, the
	// maximum number of threads supported per core by the underlying
	// processor is assumed.
	ThreadsPerCore int64 `json:",omitempty"`
}

// Nic describes a nic
// Currently only supported for Proxmox
type Nic struct {
	// IPAddress
	IPAddress string `json:",omitempty"`

	// IPv6Address
	IPv6Address string `json:",omitempty"`

	// NetMask
	NetMask string `json:",omitempty"`

	// Gateway
	Gateway string `json:",omitempty"`

	// BridgeName
	BridgeName string `json:",omitempty"`
}

// RuntimeConfig constructs runtime config
func RuntimeConfig(image string, ports []string, verbose bool) RunConfig {
	return RunConfig{ImageName: image, Ports: ports, Verbose: verbose, Memory: "2G", Accel: true}
}

// MarshalJSON ...
func (c Config) MarshalJSON() ([]byte, error) {
	var skipBaseFields []string

	if reflect.ValueOf(c.CloudConfig).IsZero() {
		skipBaseFields = append(skipBaseFields, "CloudConfig")
	}
	if reflect.ValueOf(c.RunConfig).IsZero() {
		skipBaseFields = append(skipBaseFields, "RunConfig")
	}

	type _mj Config
	cJSON, err := json.Marshal((*_mj)(&c))
	if err != nil {
		return nil, err
	}
	if len(skipBaseFields) == 0 {
		return cJSON, nil
	}

	cMap := map[string]interface{}{}
	err = json.Unmarshal(cJSON, &cMap)
	if err != nil {
		return nil, err
	}
	for _, field := range skipBaseFields {
		delete(cMap, field)
	}

	return json.Marshal(cMap)
}

// CloudVolume is an abstraction used for configuring various cloud
// based volumes.
type CloudVolume struct {
	Name       string `json:"name"`
	Iops       int64  `json:"iops"`
	Size       int64  `json:"size"`
	Throughput int64  `json:"throughput"`
	Typeof     string `json:"typeof"`
}

// IsCustom returns true if any custom root volume settings are set by
// the user.
func (cv CloudVolume) IsCustom() bool {
	if cv.Name != "" || cv.Typeof != "" || cv.Iops != 0 || cv.Throughput != 0 {
		return true
	}
	return false
}

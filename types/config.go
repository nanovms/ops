package types

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

	// Kernel
	Kernel string `json:",omitempty"`

	// Klibs host location
	KlibDir string `json:",omitempty"`

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

	// Version
	Version string `json:",omitempty"`

	// Language
	Language string `json:",omitempty"`

	// Runtime
	Runtime string `json:",omitempty"`

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

	// Platform defines the cloud provider to use with the ops CLI, currently
	// supporting aws, azure, and gcp.
	Platform string `cloud:"platform" json:",omitempty"`

	// ProjectID is used to define the project ID when the Platform is set
	// to gcp.
	ProjectID string `cloud:"projectid" json:",omitempty"`

	// SecurityGroup
	SecurityGroup string `json:",omitempty"`

	// Spot enables spot provisioning
	Spot bool `json:",omitempty"`

	// Subnet
	Subnet string `json:",omitempty"`

	// Tags
	Tags []Tag `json:",omitempty"`

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
}

// RunConfig provides runtime details
type RunConfig struct {
	// Accel defines whether hardware acceleration should be enabled.
	Accel bool `json:",omitempty"`

	// Bridged parameter is set to true if bridged networking mode is
	// in use. This also enables KVM acceleration.
	Bridged bool `json:",omitempty"`

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

	// Imagename (FIXME)
	Imagename string `json:",omitempty"`

	// InstanceGroup
	InstanceGroup string `json:",omitempty"`

	// InstanceName
	InstanceName string `json:",omitempty"`

	// IPAddress
	IPAddress string `json:",omitempty"`

	// IPv6Address
	IPv6Address string `json:",omitempty"`

	// Klibs
	Klibs []string `json:",omitempty"`

	// Memory configures the amount of memory to allocate to qemu (default
	// is 128 MiB). Optionally, a suffix of "M" or "G" can be used to
	// signify a value in megabytes or gigabytes respectively.
	Memory string `json:",omitempty"`

	// Vga whether to emulate a VGA output device
	Vga bool `json:",omitempty"`

	// Mounts
	Mounts []string `json:",omitempty"`

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
	return RunConfig{Imagename: image, Ports: ports, Verbose: verbose, Memory: "2G", Accel: true}
}

package lepton

// Config for Build
type Config struct {
	// Args defines an array of commands to execute when the image is launched.
	Args []string

	// BaseVolumeSz is an optional parameter for defining the size of the base
	// volume (defaults to the end of blocks written by TFS).
	BaseVolumeSz string

	// Boot
	Boot string

	// BuildDir
	BuildDir string

	// CloudConfig configures various attributes about the cloud provider.
	CloudConfig ProviderConfig

	// Debugflags
	Debugflags []string

	// Dirs defines an array of directory locations to include into the image.
	Dirs []string

	// Env defines a map of environment variables to specify for the image
	// runtime.
	Env map[string]string

	// Files defines an array of file locations to include into the image.
	Files []string

	// Force
	Force bool

	// Kernel
	Kernel string

	// MapDirs specifies a map of local directories to add to into the image.
	// These directory paths are then adjusted from local path specification
	// to image path specification.
	MapDirs map[string]string

	// Mounts
	Mounts map[string]string

	// NameServer is an optional parameter that defines the DNS server to use
	// for DNS resolutions (defaults to Google's DNS server: '8.8.8.8').
	NameServer string

	// NightlyBuild
	NightlyBuild bool

	// NoTrace
	NoTrace []string

	// Program
	Program string

	// ProgramPath specifies the original path of the program to refer to on
	// attach/detach.
	ProgramPath string

	// RebootOnExit defines whether the image should automatically reboot
	// if an error/failure occurs.
	RebootOnExit bool

	// RunConfig
	RunConfig RunConfig

	// TargetRoot
	TargetRoot string

	// Version
	Version string
}

// ProviderConfig give provider details
type ProviderConfig struct {
	// BucketName specifies the bucket to store the ops built image artifacts.
	BucketName string `cloud:"bucketname"`

	// Flavor
	Flavor string `cloud:"flavor"`

	// ImageName
	ImageName string `cloud:"imagename"`

	// Platform defines the cloud provider to use with the ops CLI, currently
	// supporting aws, azure, and gcp.
	Platform string `cloud:"platform"`

	// ProjectID is used to define the project ID when the Platform is set
	// to gcp.
	ProjectID string `cloud:"projectid"`

	// Zone is used to define the location of the host resource. Lists of these
	// zones are dependent on selected Platform and can be found here:
	// aws: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html
	// azure: https://azure.microsoft.com/en-us/global-infrastructure/geographies/#overview
	// gcp: https://cloud.google.com/compute/docs/regions-zones#available
	Zone string `cloud:"zone"`
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
	Accel bool

	// BaseName of the image (FIXME).
	BaseName string

	// Bridged parameter is set to true if bridged networking mode is
	// in use. This also enables KVM acceleration.
	Bridged bool

	// CPUs specifies the number of CPU cores to use
	CPUs int

	// Debug
	Debug bool

	// DomainName
	DomainName string

	// Gateway
	Gateway string

	// GdbPort
	GdbPort int

	// Imagename (FIXME)
	Imagename string

	// InstanceName
	InstanceName string

	// IPAddr
	IPAddr string

	// Klibs
	Klibs []string

	// Memory configures the amount of memory to allocate to qemu (default
	// is 128 MiB). Optionally, a suffix of "M" or "G" can be used to
	// signify a value in megabytes or gigabytes respectively.
	Memory string

	// Mounts
	Mounts []string

	// NetMask
	NetMask string

	// OnPrem is set to be true if the image is in a multi-instance/tenant
	// on-premise environment.
	OnPrem bool

	// Ports specifies a list of port to expose.
	Ports []string

	// SecurityGroup
	SecurityGroup string

	// ShowDebug
	ShowDebug bool

	// ShowErrors
	ShowErrors bool

	// ShowWarnings
	ShowWarnings bool

	// Subnet
	Subnet string

	// Tags
	Tags []Tag

	// TapName
	TapName string

	// UDP specifies if the UDP protocol is enabled (default is false).
	UDP bool

	// UDPPorts
	UDPPorts []string

	// Verbose enables logging for the runtime environment.
	Verbose bool

	// VolumeSizeInGb is an optional parameter only available for OpenStack.
	VolumeSizeInGb int

	// VPC
	VPC string
}

// RuntimeConfig constructs runtime config
func RuntimeConfig(image string, ports []string, verbose bool) RunConfig {
	return RunConfig{Imagename: image, Ports: ports, Verbose: verbose, Memory: "2G", Accel: true}
}

// NewConfig construct instance of Config with default values
func NewConfig() *Config {
	cfg := new(Config)
	cfg.RunConfig.Accel = true
	return cfg
}

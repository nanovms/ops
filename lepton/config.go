package lepton

// Config for Build
type Config struct {
	// Args defines an array of commands to execute when the image is launched.
	Args []string

	// BaseVolumeSz is an optional parameter for defining the size of the base
	// volume (defaults to the end of blocks written by TFS).
	BaseVolumeSz string

	// Boot - To be described...
	Boot string

	// BuildDir - To be described...
	BuildDir string

	// CloudConfig - To be described...
	CloudConfig ProviderConfig

	// Debugflags - To be described...
	Debugflags []string

	// Dirs defines an array of directory locations to include into the image.
	Dirs []string

	// Env defines a map of environment variables to specify for the image
	// runtime.
	Env map[string]string

	// Files defines an array of file locations to include into the image.
	Files []string

	// Force - To be described...
	Force bool

	// Kernel - To be described...
	Kernel string

	// ManifestName defines the name of the manifest file.
	ManifestName string

	// MapDirs specifies a map of local directories to add to into the image.
	// These directory paths are then adjusted from local path specification
	// to image path specification.
	MapDirs map[string]string

	// Mkfs - To be described...
	Mkfs string

	// Mounts - To be described...
	Mounts map[string]string

	// NameServer is an optional parameter that defines the DNS server to use
	// for DNS resolutions (defaults to Google's DNS server: '8.8.8.8').
	NameServer string

	// NightlyBuild - To be described...
	NightlyBuild bool

	// NoTrace - To be described...
	NoTrace []string

	// Program - To be described...
	Program string

	// ProgramPath specifies the original path of the program to refer to on
	// attach/detach.
	ProgramPath string

	// RebootOnExit defines whether the image should automatically reboot
	// if an error/failure occurs.
	RebootOnExit bool

	// RunConfig - To be described...
	RunConfig RunConfig

	// TargetRoot - To be described...
	TargetRoot string

	// Version - To be described...
	Version string
}

// ProviderConfig give provider details
type ProviderConfig struct {
	BucketName string `cloud:"bucketname"`
	Flavor     string `cloud:"flavor"`
	ImageName  string `cloud:"imagename"`
	Platform   string `cloud:"platform"`
	ProjectID  string `cloud:"projectid"`
	Zone       string `cloud:"zone"`
}

// Tag is used as property on creating instances
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RunConfig provides runtime details
type RunConfig struct {
	Accel          bool
	BaseName       string // FIXME: basename of image only
	Bridged        bool
	CPUs           int // number of cpus
	Debug          bool
	DomainName     string
	Gateway        string
	GdbPort        int
	Imagename      string // FIXME: fullpath? of image
	InstanceName   string
	IPAddr         string
	Klibs          []string
	Memory         string
	Mounts         []string
	NetMask        string
	OnPrem         bool // true if in a multi-instance/tenant on-prem env
	Ports          []string
	SecurityGroup  string
	ShowDebug      bool
	ShowErrors     bool
	ShowWarnings   bool
	Subnet         string
	Tags           []Tag
	TapName        string
	UDP            bool // enable UDP
	UDPPorts       []string
	Verbose        bool
	VolumeSizeInGb int //This option is only for openstack.
	VPC            string
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

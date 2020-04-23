package lepton

// Config for Build
type Config struct {
	Args         []string
	BuildDir     string
	Dirs         []string
	Files        []string
	MapDirs      map[string]string
	Env          map[string]string
	Debugflags   []string
	NoTrace      []string
	Program      string
	Version      string
	Boot         string
	Kernel       string
	Mkfs         string
	NameServer   string
	NightlyBuild bool
	RunConfig    RunConfig
	CloudConfig  ProviderConfig
	Force        bool
	TargetRoot   string
	BaseVolumeSz string // optional base volume sz
	ManifestName string // save manifest to
	RebootOnExit bool   // Reboot on Failure Exit
}

// ProviderConfig give provider details
type ProviderConfig struct {
	Platform   string `cloud:"platfom"`
	ProjectID  string `cloud:"projectid"`
	Zone       string `cloud:"zone"`
	BucketName string `cloud:"bucketname"`
	ImageName  string `cloud:"imagename"`
	Flavor     string `cloud:"flavor"`
}

// RunConfig provides runtime details
type RunConfig struct {
	Imagename string // FIXME: fullpath? of image
	BaseName  string // FIXME: basename of image only
	Ports     []int
	GdbPort   int
	Verbose   bool
	Memory    string
	Bridged   bool
	TapName   string
	Accel     bool
	OnPrem    bool // true if in a multi-instance/tenant on-prem env
}

// RuntimeConfig constructs runtime config
func RuntimeConfig(image string, ports []int, verbose bool) RunConfig {
	return RunConfig{Imagename: image, Ports: ports, Verbose: verbose, Memory: "2G"}
}

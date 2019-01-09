package lepton

// Build configs
type Config struct {
	Args       []string
	Dirs       []string
	Files      []string
	MapDirs    map[string]string
	Env        map[string]string
	Debugflags []string
	Program    string
	Version    string
	Boot       string
	Kernel     string
	DiskImage  string
	Mkfs       string
	NameServer string
	RunConfig  RunConfig
}

// Runtime configs
type RunConfig struct {
	Imagename string
	Ports     []int
	Verbose   bool
	Memory    string
	UserMode  bool
}

func DefaultConfig() Config {
	return Config{Boot: BootImg, Kernel: KernelImg, DiskImage: "image"}
}

func RuntimeConfig(image string, ports []int, verbose bool) RunConfig {
	return RunConfig{Imagename: image, Ports: ports, Verbose: verbose, Memory: "2G"}
}

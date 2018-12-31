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
	Boot       string
	Kernel     string
	DiskImage  string
	Mkfs       string
	NameServer string
}

// Runtime configs
type RunConfig struct {
	imagename string
	ports     []int
	verbose   bool
}

func DefaultConfig() Config {
	return Config{Boot: BootImg, Kernel: KernelImg, DiskImage: "image"}
}

func RuntimeConfig(image string, ports []int, verbose bool) RunConfig {
	return RunConfig{imagename: image, ports: ports, verbose: verbose}
}

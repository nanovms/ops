package lepton

// Config very basic config for build
type Config struct {
	Args       []string
	Dirs       []string
	Files      []string
	Debugflags []string
	Program    string
	Boot       string
	Kernel     string
	DiskImage  string
}

func DefaultConfig() *Config {
	return &Config{Boot: bootImg, Kernel: kernelImg, DiskImage: "image"}
}

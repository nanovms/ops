package lepton

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
	CreateInstance(ctx *Context) error
	ListInstances(ctx *Context) error
	GetInstances(ctx *Context) ([]CloudInstance, error)
	DeleteInstance(ctx *Context, instancename string) error
	StopInstance(ctx *Context, instancename string) error
	StartInstance(ctx *Context, instancename string) error
	GetInstanceLogs(ctx *Context, instancename string, watch bool) error
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

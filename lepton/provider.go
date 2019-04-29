package lepton

type Provider interface {
	Initialize() error
	BuildImage(ctx *Context) (string, error)
	BuildImageWithPackage(ctx *Context, pkgpath string) (string, error)
	CreateImage(ctx *Context) error
	ListImages() error
	DeleteImage(imagename string) error
	CreateInstance(ctx *Context) error
	ListInstances(ctx *Context) error
	DeleteInstance(ctx *Context, instancename string) error
	GetInstanceLogs(ctx *Context, instancename string) error
}

type Context struct {
	config   *Config
	provider *Provider
}

// NewContext Create a new context for the given provider
// valid providers are "gcp" and "onprem"
func NewContext(c *Config, provider *Provider) *Context {
	return &Context{
		config:   c,
		provider: provider,
	}
}

package lepton

type Provider interface {
	BuildImage(ctx *Context) error
	DeployImage(ctx *Context) error
	CreateInstance(ctx *Context) error
}

type Context struct {
	config   *Config
	provider *Provider
}

// Create a new context for the given provider
// valid providers are "gcp" and "onprem"
func NewContext(c *Config, provider *Provider) *Context {
	return &Context{
		config:   c,
		provider: provider,
	}
}

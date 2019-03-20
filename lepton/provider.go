package lepton

type Provider interface {
	Initialize() error
	BuildImage(ctx *Context) (string, error)
	CreateImage(ctx *Context) error
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

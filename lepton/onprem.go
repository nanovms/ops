package lepton

type OnPrem struct{}

func (p *OnPrem) BuildImage(ctx *Context) error {
	c := ctx.config
	return BuildImage(*c)
}

func (p *OnPrem) DeployImage(ctx *Context) error {
	return nil
}
func (p *OnPrem) CreateInstance(ctx *Context) error {
	return nil
}

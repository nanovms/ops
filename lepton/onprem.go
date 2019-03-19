package lepton

type OnPrem struct{}

func (p *OnPrem) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	return "", err
}

func (p *OnPrem) DeployImage(ctx *Context) error {
	return nil
}

func (p *OnPrem) CreateInstance(ctx *Context) error {
	return nil
}

func (p *OnPrem) Initialize() error {
	return nil
}

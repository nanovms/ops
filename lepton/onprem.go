package lepton

type OnPrem struct{}

// BuildImage for onprem
func (p *OnPrem) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	return "", err
}

// BuildImageWithPackage for onprem
func (p *OnPrem) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return "", nil
}

func (p *OnPrem) CreateImage(ctx *Context) error {
	return nil
}

func (p *OnPrem) CreateInstance(ctx *Context) error {
	return nil
}

func (p *OnPrem) Initialize() error {
	return nil
}

package lepton

// CreateVolume is a stub to satisfy VolumeService interface
func (v *Vultr) CreateVolume(ctx *Context, name, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (v *Vultr) GetAllVolumes(ctx *Context) (*[]NanosVolume, error) {
	return nil, nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (v *Vultr) DeleteVolume(ctx *Context, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (v *Vultr) AttachVolume(ctx *Context, image, name, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (v *Vultr) DetachVolume(ctx *Context, image, name string) error {
	return nil
}

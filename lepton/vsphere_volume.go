package lepton

// CreateVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) CreateVolume(ctx *Context, name, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (v *Vsphere) GetAllVolumes(ctx *Context) (*[]NanosVolume, error) {
	return nil, nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) DeleteVolume(ctx *Context, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) AttachVolume(ctx *Context, image, name, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) DetachVolume(ctx *Context, image, name string) error {
	return nil
}

package lepton

// CreateVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) CreateVolume(config *Config, name, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (v *Vsphere) GetAllVolumes(config *Config) error {
	return nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) DeleteVolume(config *Config, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) AttachVolume(config *Config, image, name, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) DetachVolume(config *Config, image, name string) error {
	return nil
}

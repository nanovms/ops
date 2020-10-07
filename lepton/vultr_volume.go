package lepton

// CreateVolume is a stub to satisfy VolumeService interface
func (v *Vultr) CreateVolume(config *Config, name, label, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (v *Vultr) GetAllVolumes(config *Config) error {
	return nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (v *Vultr) DeleteVolume(config *Config, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (v *Vultr) AttachVolume(config *Config, image, name, label, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (v *Vultr) DetachVolume(config *Config, image, label string) error {
	return nil
}

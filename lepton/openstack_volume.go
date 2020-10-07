package lepton

// CreateVolume is a stub to satisfy VolumeService interface
func (os *OpenStack) CreateVolume(config *Config, name, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (os *OpenStack) GetAllVolumes(config *Config) error {
	return nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (os *OpenStack) DeleteVolume(config *Config, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (os *OpenStack) AttachVolume(config *Config, image, name, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (os *OpenStack) DetachVolume(config *Config, image, name string) error {
	return nil
}

package lepton

// CreateVolume is a stub to satisfy VolumeService interface
func (a *AWS) CreateVolume(config *Config, name, label, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (a *AWS) GetAllVolumes(config *Config) error {
	return nil
}

// UpdateVolume is a stub to satisfy VolumeService interface
func (a *AWS) UpdateVolume(config *Config, name, label string) error {
	return nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (a *AWS) DeleteVolume(config *Config, name, label string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (a *AWS) AttachVolume(config *Config, image, name, label, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (a *AWS) DetachVolume(config *Config, image, label string) error {
	return nil
}

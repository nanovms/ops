package lepton

// CreateVolume is a stub to satisfy VolumeService interface
func (a *Azure) CreateVolume(config *Config, name, label, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (a *Azure) GetAllVolumes(config *Config) error {
	return nil
}

// UpdateVolume is a stub to satisfy VolumeService interface
func (a *Azure) UpdateVolume(config *Config, name, label string) error {
	return nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (a *Azure) DeleteVolume(config *Config, name, label string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (a *Azure) AttachVolume(config *Config, image, name, label, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (a *Azure) DetachVolume(config *Config, image, label string) error {
	return nil
}

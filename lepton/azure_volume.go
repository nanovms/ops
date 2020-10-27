package lepton

// CreateVolume is a stub to satisfy VolumeService interface
func (a *Azure) CreateVolume(config *Config, name, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (a *Azure) GetAllVolumes(config *Config) (*[]NanosVolume, error) {
	return nil, nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (a *Azure) DeleteVolume(config *Config, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (a *Azure) AttachVolume(config *Config, image, name, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (a *Azure) DetachVolume(config *Config, image, name string) error {
	return nil
}

package lepton

// CreateVolume is a stub to satisfy VolumeService interface
func (g *GCloud) CreateVolume(config *Config, name, label, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (g *GCloud) GetAllVolumes(config *Config) error {
	return nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (g *GCloud) DeleteVolume(config *Config, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (g *GCloud) AttachVolume(config *Config, image, name, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (g *GCloud) DetachVolume(config *Config, image, name string) error {
	return nil
}

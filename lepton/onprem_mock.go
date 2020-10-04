package lepton

// TODO unstub
type mockOnPremVolume struct{}

func (v *mockOnPremVolume) CreateVolume(config *Config, name, label, data, size, provider string) error {
	return nil
}

func (v *mockOnPremVolume) GetAllVolumes(config *Config) error {
	return nil
}

func (v *mockOnPremVolume) UpdateVolume(config *Config, name, label string) error {
	return nil
}

func (v *mockOnPremVolume) DeleteVolume(config *Config, name, label string) error {
	return nil
}

func (v *mockOnPremVolume) AttachVolume(config *Config, image, name, label, mount string) error {
	return nil
}

func (v *mockOnPremVolume) DetachVolume(config *Config, image, label string) error {
	return nil
}

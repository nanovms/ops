package crossbuild

const (
	// PackageNameNodeJS is name of NodeJS package.
	PackageNameNodeJS = "nodejs"

	// PackageNameNPM is name of NPM package.
	PackageNameNPM = "npm"
)

// InstallPackage installs OS's software package with given name.
func (env *Environment) InstallPackage(name string) error {
	if name == PackageNameNPM {
		name = PackageNameNodeJS
	}

	vmCmd := env.NewCommand("apt-get", "install", name, "-y").AsAdmin()
	if err := vmCmd.Execute(); err != nil {
		return err
	}
	return nil
}

// UninstallPackage removes OS's software package with given name.
func (env *Environment) UninstallPackage(name string) error {
	if name == PackageNameNPM {
		name = PackageNameNodeJS
	}

	vmCmd := env.NewCommand("apt-get", "purge", name, "-y").
		Then("apt-get", "autoremove", "-y").
		Then("apt-get", "clean", "-y").AsAdmin()
	if err := vmCmd.Execute(); err != nil {
		return err
	}
	return nil
}

// InstallNPMPackage installs npm package with given name.
func (env *Environment) InstallNPMPackage(name string) error {
	exists, err := env.isExecutableExists(PackageNameNPM)
	if err != nil {
		return err
	}

	if !exists {
		if err := env.InstallPackage(PackageNameNodeJS); err != nil {
			return err
		}
	}

	vmCmd := env.NewCommand("npm", "install", name, "-g", "-y").AsAdmin()
	if err := vmCmd.Execute(); err != nil {
		return err
	}
	return nil
}

// UninstallNPMPackage removes npm package with given name.
func (env *Environment) UninstallNPMPackage(name string) error {
	exists, err := env.isExecutableExists(PackageNameNPM)
	if err != nil {
		return err
	}

	if !exists {
		if err := env.InstallPackage(PackageNameNodeJS); err != nil {
			return err
		}
	}

	vmCmd := env.NewCommand("npm", "uninstall", name, "-g", "-y").AsAdmin()
	if err := vmCmd.Execute(); err != nil {
		return err
	}
	return nil
}

// Returns true if given executable exists in VM.
func (env *Environment) isExecutableExists(name string) (bool, error) {
	vmCmd := env.NewCommand("which", name).AsAdmin()
	vmCmd.SupressOutput = true
	if err := vmCmd.Execute(); err != nil {
		return false, err
	}
	return vmCmd.CombinedOutput() != "", nil
}

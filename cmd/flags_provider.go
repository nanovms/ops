package cmd

import (
	"os"

	"github.com/nanovms/ops/types"

	"github.com/go-errors/errors"
	"github.com/spf13/pflag"
)

// ProviderCommandFlags consolidates all command flags required to use a provider
type ProviderCommandFlags struct {
	TargetCloud string
	Zone        string
	Project     string
}

// MergeToConfig merge provider flags to configuration
func (flags *ProviderCommandFlags) MergeToConfig(c *types.Config) (err error) {
	if flags.TargetCloud != "" {
		c.CloudConfig.Platform = flags.TargetCloud
	}

	if c.CloudConfig.Platform == "azure" {
		c.RunConfig.Klibs = append(c.RunConfig.Klibs, "cloud_init")
	}

	if len(c.CloudConfig.Platform) == 0 {
		err = errors.New("Please select one of the cloud platform in config. [onprem, aws, gcp, do, vsphere, vultr, upcloud, hyper-v]")
		return
	}

	if flags.Zone != "" {
		c.CloudConfig.Zone = flags.Zone
	}

	if flags.Project != "" {
		c.CloudConfig.ProjectID = flags.Project
	}

	return
}

// NewProviderCommandFlags returns an instance of ProviderCommandFlags
func NewProviderCommandFlags(cmdFlags *pflag.FlagSet) (flags *ProviderCommandFlags) {
	var err error
	flags = &ProviderCommandFlags{}

	flags.TargetCloud, err = cmdFlags.GetString("target-cloud")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Project, err = cmdFlags.GetString("projectid")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Zone, err = cmdFlags.GetString("zone")
	if err != nil {
		exitWithError(err.Error())
	}

	return
}

// PersistProviderCommandFlags append a command the required flags to run an image
func PersistProviderCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.StringP("target-cloud", "t", "onprem", "cloud platform [gcp, aws, onprem, vultr, vsphere, azure, openstack, upcloud, hyper-v, oci]")
	cmdFlags.StringP("projectid", "g", os.Getenv("GOOGLE_CLOUD_PROJECT"), "project-id for GCP or set env GOOGLE_CLOUD_PROJECT")
	cmdFlags.StringP("zone", "z", os.Getenv("GOOGLE_CLOUD_ZONE"), "zone name for GCP or set env GOOGLE_CLOUD_ZONE")
}

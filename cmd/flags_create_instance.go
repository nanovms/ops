package cmd

import (
	"github.com/nanovms/ops/types"

	"github.com/spf13/pflag"
)

// CreateInstanceFlags consolidates flags used to create an instance
type CreateInstanceFlags struct {
	DomainName string
	Flavor     string
	Ports      []string
	UDPPorts   []string
}

// DeleteInstanceFlags consolidates flags used for delete an instance
type DeleteInstanceFlags struct {
	DomainName string
}

// MergeToConfig append command flags that are used to delete an instance
func (f *DeleteInstanceFlags) MergeToConfig(config *types.Config) (err error) {
	if f.DomainName != "" {
		config.CloudConfig.DomainName = f.DomainName
	}

	return nil
}

// NewDeleteInstanceCommandFlags returns an instance of DeleteInstanceFlags
func NewDeleteInstanceCommandFlags(cmdFlags *pflag.FlagSet) (flags *DeleteInstanceFlags) {
	var err error
	flags = &DeleteInstanceFlags{}

	flags.DomainName, err = cmdFlags.GetString("domainname")
	if err != nil {
		exitWithError(err.Error())
	}

	return flags
}

// MergeToConfig append command flags that are used to create an instance
func (f *CreateInstanceFlags) MergeToConfig(config *types.Config) (err error) {
	if f.DomainName != "" {
		config.CloudConfig.DomainName = f.DomainName
	}

	if f.Flavor != "" {
		config.CloudConfig.Flavor = f.Flavor
	}

	if len(f.Ports) != 0 {
		config.RunConfig.Ports = append(config.RunConfig.Ports, f.Ports...)
	}

	if len(f.UDPPorts) != 0 {
		config.RunConfig.UDPPorts = append(config.RunConfig.UDPPorts, f.UDPPorts...)
	}

	return nil
}

// NewCreateInstanceCommandFlags returns an instance of CreateInstanceFlags
func NewCreateInstanceCommandFlags(cmdFlags *pflag.FlagSet) (flags *CreateInstanceFlags) {
	var err error
	flags = &CreateInstanceFlags{}

	flags.DomainName, err = cmdFlags.GetString("domainname")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Flavor, err = cmdFlags.GetString("flavor")
	if err != nil {
		exitWithError(err.Error())
	}

	portsFlag, err := cmdFlags.GetStringArray("port")
	if err != nil {
		exitWithError(err.Error())
	}
	flags.Ports, err = PrepareNetworkPorts(portsFlag)
	if err != nil {
		exitWithError(err.Error())
	}

	udpPortsFlag, err := cmdFlags.GetStringArray("udp")
	if err != nil {
		exitWithError(err.Error())
	}
	flags.UDPPorts, err = PrepareNetworkPorts(udpPortsFlag)
	if err != nil {
		exitWithError(err.Error())
	}

	return flags
}

// PersistCreateInstanceFlags specify create instance flags in command
func PersistCreateInstanceFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.StringP("domainname", "d", "", "domain name for instance")
	cmdFlags.StringP("flavor", "f", "", "flavor name for cloud provider")
	cmdFlags.StringArrayP("port", "p", nil, "port to open")
	cmdFlags.StringArrayP("udp", "", nil, "udp ports to forward")
}

// PersistDeleteInstanceFlags specify delete instance flags in command
func PersistDeleteInstanceFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.StringP("domainname", "d", "", "domain name for instance")
}

package cmd

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/nanovms/ops/types"

	"github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
)

// BuildImageCommandFlags consolidates all command flags required to build an image in one struct
type BuildImageCommandFlags struct {
	Type            string
	CmdArgs         []string
	DisableArgsCopy bool
	CmdEnvs         []string
	ImageName       string
	Mounts          []string
	TargetRoot      string
	IPAddress       string
	IPv6Address     string
	Netmask         string
	Gateway         string
	NetConsolePort  string
	NetConsoleIP    string
	Consoles        []string
}

// MergeToConfig overrides configuration passed by argument with command flags values
func (flags *BuildImageCommandFlags) MergeToConfig(c *types.Config) (err error) {
	if len(flags.CmdEnvs) > 0 {
		if len(c.Env) == 0 {
			c.Env = make(map[string]string)
		}

		for i := 0; i < len(flags.CmdEnvs); i++ {
			ez := strings.Split(flags.CmdEnvs[i], "=")
			c.Env[ez[0]] = ez[1]
		}
	}

	if flags.TargetRoot != "" {
		c.TargetRoot = flags.TargetRoot
	}

	if flags.Type != "" {
		c.CloudConfig.ImageType = flags.Type
	}

	if flags.ImageName != "" {
		c.RunConfig.ImageName = flags.ImageName
	}

	setNanosBaseImage(c)

	if c.RunConfig.ImageName == "" && c.Program != "" {
		c.RunConfig.ImageName = c.Program
	}

	if c.RunConfig.ImageName != "" {
		imageName := strings.Split(path.Base(c.RunConfig.ImageName), ".")[0]
		if imageName == "" {
			imageName = lepton.GenerateImageName(filepath.Base(c.Program))
			c.CloudConfig.ImageName = fmt.Sprintf("%v-image", filepath.Base(c.Program))
		} else {
			c.CloudConfig.ImageName = filepath.Base(imageName)
			images := path.Join(lepton.GetOpsHome(), "images")
			imageName = path.Join(images, filepath.Base(imageName))
		}
		c.RunConfig.ImageName = imageName
	}

	if c.Args != nil {
		c.Args = append(c.Args, flags.CmdArgs...)
	} else {
		c.Args = flags.CmdArgs
	}

	if flags.Mounts != nil {
		err = lepton.AddMounts(flags.Mounts, c)
		if err != nil {
			return
		}
	}

	if len(flags.CmdArgs) > 0 {
		c.Args = flags.CmdArgs
	}

	c.DisableArgsCopy = flags.DisableArgsCopy

	if c.Program != "" {
		c.Args = append([]string{c.Program}, c.Args...)
	}

	if flags.IPAddress != "" && isIPAddressValid(flags.IPAddress) {
		c.RunConfig.IPAddress = flags.IPAddress
	}

	if flags.IPv6Address != "" {
		c.RunConfig.IPv6Address = flags.IPv6Address
	}

	if flags.Gateway != "" && isIPAddressValid(flags.Gateway) {
		c.RunConfig.Gateway = flags.Gateway
	}

	if flags.Netmask != "" && isIPAddressValid(flags.Netmask) {
		c.RunConfig.NetMask = flags.Netmask
	}

	if c.ManifestPassthrough == nil {
		c.ManifestPassthrough = make(map[string]interface{})
	}

	// we only set the netconsole port and ip if the consoles have values
	if len(flags.Consoles) > 0 {
		c.ManifestPassthrough["netconsole_port"] = flags.NetConsolePort
		c.ManifestPassthrough["netconsole_ip"] = flags.NetConsoleIP
		c.ManifestPassthrough["consoles"] = flags.Consoles
	}
	return
}

// NewBuildImageCommandFlags returns an instance of BuildImageCommandFlags initialized with command flags values
func NewBuildImageCommandFlags(cmdFlags *pflag.FlagSet) (flags *BuildImageCommandFlags) {
	var err error
	flags = &BuildImageCommandFlags{}

	flags.Type, err = cmdFlags.GetString("type")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.CmdEnvs, err = cmdFlags.GetStringArray("envs")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.ImageName, err = cmdFlags.GetString("imagename")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.TargetRoot, err = cmdFlags.GetString("target-root")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Mounts, err = cmdFlags.GetStringArray("mounts")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.CmdArgs, err = cmdFlags.GetStringArray("args")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.DisableArgsCopy, err = cmdFlags.GetBool("disable-args-copy")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Gateway, err = cmdFlags.GetString("gateway")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.IPAddress, err = cmdFlags.GetString("ip-address")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.IPv6Address, err = cmdFlags.GetString("ipv6-address")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Netmask, err = cmdFlags.GetString("netmask")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.NetConsolePort, err = cmdFlags.GetString("netconsole-port")
	if err != nil {
		exitWithError(err.Error())
	}
	flags.NetConsoleIP, err = cmdFlags.GetString("netconsole-ip")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Consoles, err = cmdFlags.GetStringArray("consoles")
	if err != nil {
		exitWithError(err.Error())
	}
	return
}

// PersistBuildImageCommandFlags append a command the required flags to run an image
func PersistBuildImageCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.String("type", "", "image type (target platform-specific)")
	cmdFlags.StringArrayP("envs", "e", nil, "env arguments")
	cmdFlags.StringP("target-root", "r", "", "target root")
	cmdFlags.StringP("imagename", "i", "", "image name")
	cmdFlags.StringArray("mounts", nil, "mount <volume_id:mount_path>")
	cmdFlags.StringArrayP("args", "a", nil, "command line arguments")
	cmdFlags.BoolP("disable-args-copy", "", false, "disable copying of files passed as arguments")
	cmdFlags.String("ip-address", "", "static ip address")
	cmdFlags.String("ipv6-address", "", "static ipv6 address")
	cmdFlags.String("gateway", "", "network gateway")
	cmdFlags.String("netmask", "255.255.255.0", "network mask")
	cmdFlags.StringP("netconsole-port", "", "4444", "set net console port")
	cmdFlags.StringP("netconsole-ip", "", "10.0.2.2", "set net console ip")
	cmdFlags.StringArrayP("consoles", "", []string{}, "set different consoles to forward logs to")
}

func setNanosBaseImage(c *types.Config) {
	var err error
	var currversion string

	if c.NightlyBuild {
		currversion, err = downloadNightlyImages(c)
	} else if c.NanosVersion != "" && c.NanosVersion != "0.0" {
		currversion = c.NanosVersion
	} else {
		currversion, err = getCurrentVersion()
	}

	panicOnError(err)
	updateNanosToolsPaths(c, currversion)
}

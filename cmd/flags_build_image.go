package cmd

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/nanovms/ops/config"
	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
)

// BuildImageCommandFlags consolidates all command flags required to build an image in one struct
type BuildImageCommandFlags struct {
	CmdArgs    []string
	CmdEnvs    []string
	ImageName  string
	Mounts     []string
	TargetRoot string
}

// MergeToConfig overrides configuration passed by argument with command flags values
func (flags *BuildImageCommandFlags) MergeToConfig(c *config.Config) (err error) {
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

	if flags.ImageName != "" {
		c.RunConfig.Imagename = flags.ImageName
	}

	setNanosBaseImage(c)

	if len(flags.CmdArgs) != 0 {
		c.Program = flags.CmdArgs[0]
	} else if len(c.Args) != 0 {
		c.Program = c.Args[0]
	}

	if c.RunConfig.Imagename == "" && c.Program != "" {
		c.RunConfig.Imagename = c.Program
	}

	if c.RunConfig.Imagename != "" {
		imageName := c.RunConfig.Imagename
		if imageName == "" {
			imageName = lepton.GenerateImageName(c.Program)
			c.CloudConfig.ImageName = fmt.Sprintf("%v-image", filepath.Base(c.Program))
		} else {
			c.CloudConfig.ImageName = imageName
			images := path.Join(lepton.GetOpsHome(), "images")
			imageName = path.Join(images, filepath.Base(imageName)+".img")
		}
		c.RunConfig.Imagename = imageName
	}

	if c.Args != nil {
		c.Args = append(c.Args, flags.CmdArgs...)
	} else {
		c.Args = flags.CmdArgs
	}

	if flags.Mounts != nil {
		// borrow BuildDir from config
		bd := c.BuildDir
		c.BuildDir = api.LocalVolumeDir
		err = api.AddMounts(flags.Mounts, c)
		if err != nil {
			exitWithError(err.Error())
		}
		c.BuildDir = bd
	}

	if len(flags.CmdArgs) > 0 {
		c.Args = flags.CmdArgs
	}

	return
}

// NewBuildImageCommandFlags returns an instance of BuildImageCommandFlags initialized with command flags values
func NewBuildImageCommandFlags(cmdFlags *pflag.FlagSet) (flags *BuildImageCommandFlags) {
	var err error
	flags = &BuildImageCommandFlags{}

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

	return
}

// PersistBuildImageCommandFlags append a command the required flags to run an image
func PersistBuildImageCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.StringArrayP("envs", "e", nil, "env arguments")
	cmdFlags.StringP("target-root", "r", "", "target root")
	cmdFlags.StringP("imagename", "i", "", "image name")
	cmdFlags.StringArray("mounts", nil, "mount <volume_id:mount_path>")
	cmdFlags.StringArrayP("args", "a", nil, "command line arguments")
}

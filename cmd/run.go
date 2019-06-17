package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func buildImages(c *api.Config) {
	prepareImages(c)
	err := api.BuildImage(*c)
	panicOnError(err)
}

func runCommandHandler(cmd *cobra.Command, args []string) {
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}

	force, err := strconv.ParseBool(cmd.Flag("force").Value.String())
	if err != nil {
		panic(err)
	}

	nightly, err := strconv.ParseBool(cmd.Flag("nightly").Value.String())
	if err != nil {
		panic(err)
	}

	debugflags, err := strconv.ParseBool(cmd.Flag("debug").Value.String())
	if err != nil {
		panic(err)
	}

	noTrace, err := cmd.Flags().GetStringArray("no-trace")
	if err != nil {
		panic(err)
	}

	verbose, err := strconv.ParseBool(cmd.Flag("verbose").Value.String())
	if err != nil {
		panic(err)
	}

	bridged, err := strconv.ParseBool(cmd.Flag("bridged").Value.String())
	if err != nil {
		panic(err)
	}

	skipbuild, err := strconv.ParseBool(cmd.Flag("skipbuild").Value.String())
	if err != nil {
		panic(err)
	}

	targetRoot, err := cmd.Flags().GetString("target-root")
	if err != nil {
		panic(err)
	}

	accel, err := cmd.Flags().GetBool("accel")
	if err != nil {
		panic(err)
	}

	manifestName, err := cmd.Flags().GetString("manifest-name")
	if err != nil {
		panic(err)
	}

	tapDeviceName, err := cmd.Flags().GetString("tapname")
	if err != nil {
		panic(err)
	}

	config, _ := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}
	config = strings.TrimSpace(config)

	cmdargs, err := cmd.Flags().GetStringArray("args")
	if err != nil {
		panic(err)
	}

	c := unWarpConfig(config)
	c.Args = append(c.Args, cmdargs...)
	c.Program = args[0]
	if debugflags {
		c.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}
	c.TargetRoot = targetRoot

	c.RunConfig.TapName = tapDeviceName
	c.RunConfig.Verbose = verbose
	c.RunConfig.Bridged = bridged
	c.RunConfig.Accel = accel
	c.NightlyBuild = nightly
	c.Force = force
	c.ManifestName = manifestName
	if len(noTrace) > 0 {
		c.NoTrace = noTrace
	}
	setDefaultImageName(cmd, c)

	if !skipbuild {
		buildImages(c)
	}

	ports := []int{}
	port, err := cmd.Flags().GetStringArray("port")

	if err != nil {
		panic(err)
	}
	for _, p := range port {
		i, err := strconv.Atoi(p)
		if err != nil {
			panic(err)
		}
		ports = append(ports, i)
	}

	fmt.Printf("booting %s ...\n", c.RunConfig.Imagename)

	initDefaultRunConfigs(c, ports)
	hypervisor.Start(&c.RunConfig)
}

// RunCommand provides support for running binary with nanos
func RunCommand() *cobra.Command {
	var ports []string
	var force bool
	var debugflags bool
	var noTrace []string
	var args []string
	var verbose bool
	var bridged bool
	var nightly bool
	var tap string

	var skipbuild bool
	var manifestName string
	var accel bool
	var config string
	var imageName string
	var targetRoot string

	var cmdRun = &cobra.Command{
		Use:   "run [elf]",
		Short: "Run ELF binary as unikernel",
		Args:  cobra.MinimumNArgs(1),
		Run:   runCommandHandler,
	}
	cmdRun.PersistentFlags().StringArrayVarP(&ports, "port", "p", nil, "port to forward")
	cmdRun.PersistentFlags().BoolVarP(&force, "force", "f", false, "update images")
	cmdRun.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")
	cmdRun.PersistentFlags().BoolVarP(&debugflags, "debug", "d", false, "enable all debug flags")
	cmdRun.PersistentFlags().StringArrayVarP(&noTrace, "no-trace", "", nil, "do not trace syscall")
	cmdRun.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "command line arguments")
	cmdRun.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdRun.PersistentFlags().StringVarP(&targetRoot, "target-root", "r", "", "target root")
	cmdRun.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	cmdRun.PersistentFlags().BoolVarP(&bridged, "bridged", "b", false, "bridge networking")
	cmdRun.PersistentFlags().StringVarP(&tap, "tapname", "t", "tap0", "tap device name")
	cmdRun.PersistentFlags().BoolVarP(&skipbuild, "skipbuild", "s", false, "skip building image")
	cmdRun.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	cmdRun.PersistentFlags().StringVarP(&manifestName, "manifest-name", "m", "", "save manifest to file")
	cmdRun.PersistentFlags().BoolVarP(&accel, "accel", "x", false, "use cpu virtualization extension")

	return cmdRun
}

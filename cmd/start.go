package cmd

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func startCommandHandler(cmd *cobra.Command, args []string) {

	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	if config == "" {
		fmt.Fprintf(os.Stderr, "Invalid config specified: %q\n", config)
		os.Exit(1)
	}
	c := unWarpConfig(config)

	local, err := cmd.Flags().GetBool("local")
	if err != nil {
		panic(err)
	}

	var expackage string
	if local {
		expackage = path.Join(api.GetOpsHome(), "local_packages", c.PackageName)
	} else {
		expackage = downloadAndExtractPackage(c.PackageName)
	}

	// load the package manifest
	manifest := path.Join(expackage, "package.manifest")
	if _, err := os.Stat(manifest); err != nil {
		panic(err)
	}

	pkgConfig := unWarpConfig(manifest)

	debugflags, err := strconv.ParseBool(cmd.Flag("debug").Value.String())
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

	force, err := strconv.ParseBool(cmd.Flag("force").Value.String())
	if err != nil {
		panic(err)
	}

	nightly, err := strconv.ParseBool(cmd.Flag("nightly").Value.String())
	if err != nil {
		panic(err)
	}

	accel, err := cmd.Flags().GetBool("accel")
	if err != nil {
		panic(err)
	}

	skipbuild, err := strconv.ParseBool(cmd.Flag("skipbuild").Value.String())
	if err != nil {
		panic(err)
	}

	cmdargs, _ := cmd.Flags().GetStringArray("args")
	c.Args = append(c.Args, cmdargs...)

	if debugflags {
		pkgConfig.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}

	c = mergeConfigs(pkgConfig, c)
	pkgConfig.RunConfig.Verbose = verbose
	pkgConfig.RunConfig.Bridged = bridged
	pkgConfig.NightlyBuild = nightly
	pkgConfig.Force = force
	pkgConfig.RunConfig.Accel = accel
	setDefaultImageName(cmd, c)

	if !skipbuild {
		if err = buildFromPackage(expackage, c); err != nil {
			panic(err)
		}
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

// LoadCommand helps you to run application with package
func StartCommand() *cobra.Command {
	var (
		ports, args                    []string
		force, debugflags, verbose     bool
		nightly, accel, bridged, local bool
		skipbuild                      bool
		config, imageName              string
	)

	var cmdStartConfig = &cobra.Command{
		Use:   "start -c [config.json]",
		Short: "Start a new instance using the config",
		Run:   startCommandHandler,
	}
	cmdStartConfig.PersistentFlags().StringArrayVarP(&ports, "port", "p", nil, "port to forward")
	cmdStartConfig.PersistentFlags().BoolVarP(&force, "force", "f", false, "update images")
	cmdStartConfig.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")
	cmdStartConfig.PersistentFlags().BoolVarP(&debugflags, "debug", "d", false, "enable all debug flags")
	cmdStartConfig.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "command line arguments")
	cmdStartConfig.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file (required)")
	cmdStartConfig.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	cmdStartConfig.PersistentFlags().BoolVarP(&bridged, "bridged", "b", false, "bridge networking")
	cmdStartConfig.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	cmdStartConfig.PersistentFlags().BoolVarP(&accel, "accel", "x", false, "use cpu virtualization extension")
	cmdStartConfig.PersistentFlags().BoolVarP(&skipbuild, "skipbuild", "s", false, "skip building package image")
	cmdStartConfig.PersistentFlags().BoolVarP(&local, "local", "l", false, "load local package")
	cmdStartConfig.MarkPersistentFlagRequired("config")
	return cmdStartConfig
}

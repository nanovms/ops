package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/network"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
)

func parseVersion(s string, width int) int64 {
	strList := strings.Split(s, ".")
	format := fmt.Sprintf("%%s%%0%ds", width)
	v := ""
	for _, value := range strList {
		v = fmt.Sprintf(format, v, value)
	}
	var result int64
	var err error
	if result, err = strconv.ParseInt(v, 10, 64); err != nil {
		panic(err)
	}
	return result
}

func downloadReleaseImages() (string, error) {
	var err error

	// if it's first run or we have an update
	local, remote := api.LocalReleaseVersion, api.LatestReleaseVersion
	if local == "0.0" {
		err = api.DownloadReleaseImages(remote)
		if err != nil {
			return "", err
		}
		return remote, nil

	}

	if parseVersion(local, 4) != parseVersion(remote, 4) {
		fmt.Println(chalk.Red, "You are running an older version of Ops.", chalk.Reset)
		fmt.Println(chalk.Red, "Update: Run", chalk.Reset, chalk.Bold.TextStyle("`ops update`"))
	}

	return local, nil
}

func downloadNightlyImages(c *api.Config) (string, error) {
	var err error
	err = api.DownloadNightlyImages(c)
	return "nightly", err
}

// merge userconfig to package config, user config takes precedence
func mergeConfigs(pkgConfig *api.Config, usrConfig *api.Config) *api.Config {

	pkgConfig.Args = append(pkgConfig.Args, usrConfig.Args...)
	pkgConfig.Dirs = append(pkgConfig.Dirs, usrConfig.Dirs...)
	pkgConfig.Files = append(pkgConfig.Files, usrConfig.Files...)

	if pkgConfig.MapDirs == nil {
		pkgConfig.MapDirs = make(map[string]string)
	}

	if pkgConfig.Env == nil {
		pkgConfig.Env = make(map[string]string)
	}

	for k, v := range usrConfig.MapDirs {
		pkgConfig.MapDirs[k] = v
	}

	for k, v := range usrConfig.Env {
		pkgConfig.Env[k] = v
	}

	pkgConfig.BaseVolumeSz = usrConfig.BaseVolumeSz
	pkgConfig.RunConfig = usrConfig.RunConfig
	pkgConfig.CloudConfig = usrConfig.CloudConfig
	pkgConfig.Kernel = usrConfig.Kernel
	pkgConfig.Boot = usrConfig.Boot
	pkgConfig.TargetRoot = usrConfig.TargetRoot
	pkgConfig.Force = usrConfig.Force
	pkgConfig.NightlyBuild = usrConfig.NightlyBuild
	pkgConfig.NameServer = usrConfig.NameServer

	return pkgConfig
}

func buildFromPackage(packagepath string, c *api.Config) error {
	var err error
	var currversion string

	if c.NightlyBuild {
		currversion, err = downloadNightlyImages(c)
	} else {
		currversion, err = downloadReleaseImages()
	}
	panicOnError(err)
	fixupConfigImages(c, currversion)
	return api.BuildImageFromPackage(packagepath, *c)
}

func loadCommandHandler(cmd *cobra.Command, args []string) {
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}

	local, err := cmd.Flags().GetBool("local")
	if err != nil {
		panic(err)
	}

	var expackage string
	if local {
		expackage = path.Join(api.GetOpsHome(), "local_packages", args[0])
	} else {
		expackage = downloadAndExtractPackage(args[0])
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

	tapDeviceName, err := cmd.Flags().GetString("tapname")
	if err != nil {
		panic(err)
	}

	ipaddr, err := cmd.Flags().GetString("ip-address")
	if err != nil {
		panic(err)
	}

	gateway, err := cmd.Flags().GetString("gateway")
	if err != nil {
		panic(err)
	}

	netmask, err := cmd.Flags().GetString("netmask")
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

	syscallSummary, err := strconv.ParseBool(cmd.Flag("syscall-summary").Value.String())
	if err != nil {
		panic(err)
	}

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	cmdargs, _ := cmd.Flags().GetStringArray("args")
	c := unWarpConfig(config)
	c.Args = append(c.Args, cmdargs...)
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), c)

	if debugflags {
		pkgConfig.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}

	if syscallSummary {
		pkgConfig.Debugflags = append(pkgConfig.Debugflags, "syscall_summary")
	}

	c = mergeConfigs(pkgConfig, c)
	pkgConfig.RunConfig.Verbose = verbose
	pkgConfig.RunConfig.Bridged = bridged
	pkgConfig.RunConfig.TapName = tapDeviceName
	pkgConfig.NightlyBuild = nightly
	pkgConfig.Force = force
	pkgConfig.RunConfig.Accel = accel

	if ipaddr != "" && isIPAddressValid(ipaddr) {
		c.RunConfig.IPAddr = ipaddr

		if gateway == "" || !isIPAddressValid(gateway) {
			// assumes the default gateway is the first IP in the network range
			ip := net.ParseIP(ipaddr).To4()
			ip[3] = byte(1)
			c.RunConfig.Gateway = ip.String()
		} else {
			c.RunConfig.Gateway = gateway
		}

		if netmask != "" && !isIPAddressValid(netmask) {
			c.RunConfig.NetMask = "255.255.255.0"
		} else {
			c.RunConfig.NetMask = netmask
		}
	}

	setDefaultImageName(cmd, c)

	mounts, _ := cmd.Flags().GetStringArray("mounts")
	// borrow BuildDir from config
	bd := c.BuildDir
	c.BuildDir = api.LocalVolumeDir
	err = api.AddMounts(mounts, c)
	if err != nil {
		log.Fatal(err)
	}
	c.BuildDir = bd

	if !skipbuild {
		if err = buildFromPackage(expackage, c); err != nil {
			panic(err)
		}
	}

	portsFlag, err := cmd.Flags().GetStringArray("port")
	if err != nil {
		panic(err)
	}
	ports, err := prepareNetworkPorts(portsFlag)
	if err != nil {
		exitWithError(err.Error())
		return
	}

	networkService := network.NewIprouteNetworkService()

	if tapDeviceName != "" {
		err := network.SetupNetworkInterfaces(networkService, tapDeviceName, bridged, ipaddr, netmask)
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("booting %s ...\n", c.RunConfig.Imagename)
	initDefaultRunConfigs(c, ports)
	hypervisor.Start(&c.RunConfig)

	if tapDeviceName != "" {
		err := network.TurnOffNetworkInterfaces(networkService, tapDeviceName, bridged, bridgeName)
		if err != nil {
			panic(err)
		}
	}
}

// LoadCommand helps you to run application with package
func LoadCommand() *cobra.Command {
	var (
		ports, args, mounts            []string
		force, debugflags, verbose     bool
		nightly, accel, bridged, local bool
		skipbuild, syscallSummary      bool
		config, imageName              string
	)

	var cmdLoadPackage = &cobra.Command{
		Use:   "load [packagename]",
		Short: "load and run a package from ['ops pkg list']",
		Args:  cobra.MinimumNArgs(1),
		Run:   loadCommandHandler,
	}
	cmdLoadPackage.PersistentFlags().StringArrayVarP(&ports, "port", "p", nil, "port to forward")
	cmdLoadPackage.PersistentFlags().BoolVarP(&force, "force", "f", false, "update images")
	cmdLoadPackage.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")
	cmdLoadPackage.PersistentFlags().BoolVarP(&debugflags, "debug", "d", false, "enable all debug flags")
	cmdLoadPackage.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "command line arguments")
	cmdLoadPackage.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdLoadPackage.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	cmdLoadPackage.PersistentFlags().BoolVarP(&bridged, "bridged", "b", false, "bridge networking")
	cmdLoadPackage.PersistentFlags().StringP("tapname", "t", "", "tap device name")
	cmdLoadPackage.PersistentFlags().String("ip-address", "", "static ip address")
	cmdLoadPackage.PersistentFlags().String("gateway", "", "network gateway")
	cmdLoadPackage.PersistentFlags().String("netmask", "255.255.255.0", "network mask")
	cmdLoadPackage.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	cmdLoadPackage.PersistentFlags().BoolVar(&accel, "accel", true, "use cpu virtualization extension")
	cmdLoadPackage.PersistentFlags().BoolVarP(&skipbuild, "skipbuild", "s", false, "skip building package image")
	cmdLoadPackage.PersistentFlags().BoolVarP(&local, "local", "l", false, "load local package")
	cmdLoadPackage.PersistentFlags().StringArrayVar(&mounts, "mounts", nil, "<volume_id/label>:/<mount_path>")
	cmdLoadPackage.PersistentFlags().BoolVar(&syscallSummary, "syscall-summary", false, "print syscall summary on exit")

	return cmdLoadPackage
}

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
)

func buildImages(c *api.Config) error {
	prepareImages(c)
	err := api.BuildImage(*c)
	if err != nil {
		return err
	}
	return nil
}

func runCommandHandler(cmd *cobra.Command, args []string) {
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		fmt.Println("No hypervisor found on $PATH")
		fmt.Println("Please install OPS using curl https://ops.city/get.sh -sSfL | sh")
		os.Exit(1)
	}

	force, err := strconv.ParseBool(cmd.Flag("force").Value.String())
	if err != nil {
		panic(err)
	}

	nightly, err := strconv.ParseBool(cmd.Flag("nightly").Value.String())
	if err != nil {
		panic(err)
	}

	debug, err := strconv.ParseBool(cmd.Flag("debug").Value.String())
	if err != nil {
		panic(err)
	}

	trace, err := strconv.ParseBool(cmd.Flag("trace").Value.String())
	if err != nil {
		panic(err)
	}

	gdbport, err := cmd.Flags().GetInt("gdbport")
	if err != nil {
		panic(err)
	}

	smp, err := cmd.Flags().GetInt("smp")
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

	var accel bool
	if debug {
		accel = false
	} else {
		accel, err = cmd.Flags().GetBool("accel")
		if err != nil {
			panic(err)
		}
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

	config, _ := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}
	config = strings.TrimSpace(config)

	cmdargs, err := cmd.Flags().GetStringArray("args")
	if err != nil {
		panic(err)
	}

	cmdenvs, err := cmd.Flags().GetStringArray("envs")
	if err != nil {
		panic(err)
	}

	mounts, err := cmd.Flags().GetStringArray("mounts")
	if err != nil {
		panic(err)
	}

	syscallSummary, err := cmd.Flags().GetBool("syscall-summary")
	if err != nil {
		panic(err)
	}

	c := unWarpConfig(config)
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), c)

	c.Program = args[0]
	curdir, _ := os.Getwd()
	c.ProgramPath = path.Join(curdir, args[0])

	if len(c.Args) == 0 {
		c.Args = append([]string{args[0]}, cmdargs...)
	} else {
		c.Args = append(c.Args, cmdargs...)
	}

	if len(cmdenvs) > 0 {
		if len(c.Env) == 0 {
			c.Env = make(map[string]string)
		}

		for i := 0; i < len(cmdenvs); i++ {
			ez := strings.Split(cmdenvs[i], "=")
			c.Env[ez[0]] = ez[1]
		}
	}

	c.Debugflags = []string{}

	if trace {
		c.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}

	if debug {
		c.RunConfig.Debug = true

		c.Debugflags = append(c.Debugflags, "noaslr")

		elfFile, err := api.GetElfFileInfo(c.ProgramPath)
		if err != nil {
			log.Fatal(err)
		}

		if api.IsDynamicLinked(elfFile) {
			log.Fatalf("Program %s must be linked statically", c.ProgramPath)
		}

		if !api.HasDebuggingSymbols(elfFile) {
			log.Fatalf("Program %s must be compiled with debugging symbols", c.ProgramPath)
		}
	}

	if syscallSummary {
		c.Debugflags = append(c.Debugflags, "syscall_summary")
	}

	c.RunConfig.GdbPort = gdbport

	if smp > 0 {
		c.RunConfig.CPUs = smp
	}

	c.TargetRoot = targetRoot

	c.RunConfig.TapName = tapDeviceName
	c.RunConfig.Verbose = verbose
	c.RunConfig.Bridged = bridged
	c.RunConfig.Accel = accel
	c.NightlyBuild = nightly
	c.Force = force

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

	if len(noTrace) > 0 {
		c.NoTrace = noTrace
	}
	setDefaultImageName(cmd, c)

	// borrow BuildDir from config
	bd := c.BuildDir
	c.BuildDir = api.LocalVolumeDir
	err = api.AddMounts(mounts, c)
	if err != nil {
		log.Fatal(err)
	}
	c.BuildDir = bd

	if !skipbuild {
		err = buildImages(c)
		if err != nil {
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

	for _, p := range ports {
		i, err := strconv.Atoi(p)
		if err == nil && i == gdbport {
			errstr := fmt.Sprintf("Port %d is forwarded and cannot be used as gdb port", gdbport)
			panic(errors.New(errstr))
		}
	}

	fmt.Printf("booting %s ...\n", c.RunConfig.Imagename)

	networkService := network.NewIprouteNetworkService()

	if tapDeviceName != "" {
		err := network.SetupNetworkInterfaces(networkService, tapDeviceName, bridged, ipaddr, netmask)
		if err != nil {
			panic(err)
		}
	}

	initDefaultRunConfigs(c, ports)
	hypervisor.Start(&c.RunConfig)

	if tapDeviceName != "" {
		err := network.TurnOffNetworkInterfaces(networkService, tapDeviceName, bridged, bridgeName)
		if err != nil {
			panic(err)
		}
	}
}

// RunCommand provides support for running binary with nanos
func RunCommand() *cobra.Command {
	var ports []string
	var force bool
	var debug bool
	var trace bool
	var gdbport int
	var smp int
	var noTrace []string
	var args []string
	var envs []string
	var verbose bool
	var bridged bool
	var nightly bool
	var tap string
	var mounts []string
	var syscallSummary bool

	var skipbuild bool
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
	cmdRun.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable interactive debugger")
	cmdRun.PersistentFlags().BoolVarP(&trace, "trace", "", false, "enable required flags to trace")
	cmdRun.PersistentFlags().IntVarP(&gdbport, "gdbport", "g", 0, "qemu TCP port used for GDB interface")
	cmdRun.PersistentFlags().StringArrayVarP(&noTrace, "no-trace", "", nil, "do not trace syscall")
	cmdRun.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "command line arguments")
	cmdRun.PersistentFlags().StringArrayVarP(&envs, "envs", "e", nil, "env arguments")
	cmdRun.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdRun.PersistentFlags().StringVarP(&targetRoot, "target-root", "r", "", "target root")
	cmdRun.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	cmdRun.PersistentFlags().BoolVarP(&bridged, "bridged", "b", false, "bridge networking")
	cmdRun.PersistentFlags().StringVarP(&tap, "tapname", "t", "", "tap device name")
	cmdRun.PersistentFlags().String("ip-address", "", "static ip address")
	cmdRun.PersistentFlags().String("gateway", "", "network gateway")
	cmdRun.PersistentFlags().String("netmask", "255.255.255.0", "network mask")
	cmdRun.PersistentFlags().BoolVarP(&skipbuild, "skipbuild", "s", false, "skip building image")
	cmdRun.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	cmdRun.PersistentFlags().BoolVar(&accel, "accel", true, "use cpu virtualization extension")
	cmdRun.PersistentFlags().IntVarP(&smp, "smp", "", 1, "number of threads to use")
	cmdRun.PersistentFlags().StringArrayVar(&mounts, "mounts", nil, "<volume_id/label>:/<mount_path>")
	cmdRun.PersistentFlags().BoolVar(&syscallSummary, "syscall-summary", false, "print syscall summary on exit")

	return cmdRun
}

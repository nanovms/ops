package cmd

import (
	"debug/elf"
	"fmt"
	"net"
	"strconv"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/config"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
)

// RunLocalInstanceCommandFlags consolidates all command flags required to run a local instance in one struct
type RunLocalInstanceCommandFlags struct {
	Accel          bool
	Bridged        bool
	BridgeName     string
	Debug          bool
	Force          bool
	Gateway        string
	GDBPort        int
	IPAddress      string
	Netmask        string
	NoTrace        []string
	Ports          []string
	SkipBuild      bool
	Smp            int
	SyscallSummary bool
	TapName        string
	Trace          bool
	Verbose        bool
}

// MergeToConfig overrides configuration passed by argument with command flags values
func (flags *RunLocalInstanceCommandFlags) MergeToConfig(c *config.Config) (err error) {
	c.Debugflags = []string{}

	if flags.Trace {
		c.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}

	if flags.Debug {
		c.RunConfig.Debug = true

		c.Debugflags = append(c.Debugflags, "noaslr")

		var elfFile *elf.File

		elfFile, err = api.GetElfFileInfo(c.ProgramPath)
		if err != nil {
			return
		}

		if api.IsDynamicLinked(elfFile) {
			return fmt.Errorf("Program %s must be linked statically", c.ProgramPath)
		}

		if !api.HasDebuggingSymbols(elfFile) {
			return fmt.Errorf("Program %s must be compiled with debugging symbols", c.ProgramPath)
		}
	}

	if flags.SyscallSummary {
		c.Debugflags = append(c.Debugflags, "syscall_summary")
	}

	if flags.Smp > 0 {
		c.RunConfig.CPUs = flags.Smp
	}

	if flags.GDBPort != 0 {
		c.RunConfig.GdbPort = flags.GDBPort
	}

	if flags.TapName != "" {
		c.RunConfig.TapName = flags.TapName
	}

	if flags.BridgeName != "" {
		c.RunConfig.BridgeName = flags.BridgeName
	}

	c.RunConfig.Verbose = flags.Verbose
	c.RunConfig.Bridged = flags.Bridged
	c.RunConfig.Accel = flags.Accel
	c.Force = flags.Force

	ipaddr := flags.IPAddress
	gateway := flags.Gateway
	netmask := flags.Netmask
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

	if len(flags.NoTrace) > 0 {
		c.NoTrace = flags.NoTrace
	}

	ports, err := PrepareNetworkPorts(flags.Ports)
	if err != nil {
		exitWithError(err.Error())
		return
	}

	for _, p := range ports {
		i, err := strconv.Atoi(p)
		if err == nil && i == flags.GDBPort {
			errstr := fmt.Sprintf("Port %d is forwarded and cannot be used as gdb port", flags.GDBPort)
			return errors.New(errstr)
		}
	}

	c.RunConfig.Ports = append(c.RunConfig.Ports, ports...)

	return
}

// NewRunLocalInstanceCommandFlags returns an instance of RunLocalInstanceCommandFlags initialized with command flags values
func NewRunLocalInstanceCommandFlags(cmdFlags *pflag.FlagSet) (flags *RunLocalInstanceCommandFlags) {
	var err error
	flags = &RunLocalInstanceCommandFlags{}

	flags.Bridged, err = cmdFlags.GetBool("bridged")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Debug, err = cmdFlags.GetBool("debug")
	if err != nil {
		exitWithError(err.Error())
	}

	if flags.Debug {
		flags.Accel = false
	} else {
		flags.Accel, err = cmdFlags.GetBool("accel")
		if err != nil {
			exitWithError(err.Error())
		}
	}

	flags.Force, err = cmdFlags.GetBool("force")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Gateway, err = cmdFlags.GetString("gateway")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.GDBPort, err = cmdFlags.GetInt("gdbport")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.IPAddress, err = cmdFlags.GetString("ip-address")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Netmask, err = cmdFlags.GetString("netmask")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.NoTrace, err = cmdFlags.GetStringArray("no-trace")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Ports, err = cmdFlags.GetStringArray("port")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.SkipBuild, err = cmdFlags.GetBool("skipbuild")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Smp, err = cmdFlags.GetInt("smp")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.SyscallSummary, err = cmdFlags.GetBool("syscall-summary")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.TapName, err = cmdFlags.GetString("tapname")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.BridgeName, err = cmdFlags.GetString("bridgename")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Trace, err = cmdFlags.GetBool("trace")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Verbose, err = cmdFlags.GetBool("verbose")
	if err != nil {
		exitWithError(err.Error())
	}

	return
}

// PersistRunLocalInstanceCommandFlags append a command the required flags to run an image
func PersistRunLocalInstanceCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.StringArrayP("port", "p", nil, "port to forward")
	cmdFlags.BoolP("force", "f", false, "update images")
	cmdFlags.BoolP("debug", "d", false, "enable interactive debugger")
	cmdFlags.BoolP("trace", "", false, "enable required flags to trace")
	cmdFlags.IntP("gdbport", "g", 0, "qemu TCP port used for GDB interface")
	cmdFlags.StringArrayP("no-trace", "", nil, "do not trace syscall")
	cmdFlags.BoolP("verbose", "v", false, "verbose")
	cmdFlags.BoolP("bridged", "b", false, "bridge networking")
	cmdFlags.StringP("bridgename", "", "", "bridge name")
	cmdFlags.StringP("tapname", "t", "", "tap device name")
	cmdFlags.String("ip-address", "", "static ip address")
	cmdFlags.String("gateway", "", "network gateway")
	cmdFlags.String("netmask", "255.255.255.0", "network mask")
	cmdFlags.BoolP("skipbuild", "s", false, "skip building image")
	cmdFlags.Bool("accel", true, "use cpu virtualization extension")
	cmdFlags.IntP("smp", "", 1, "number of threads to use")
	cmdFlags.Bool("syscall-summary", false, "print syscall summary on exit")
}

// isIPAddressValid checks whether IP address is valid
func isIPAddressValid(ip string) bool {
	if net.ParseIP(ip) == nil {
		return false
	}

	return true
}

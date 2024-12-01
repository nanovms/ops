package cmd

import (
	"debug/elf"
	"fmt"
	"net"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"

	"github.com/go-errors/errors"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
)

// RunLocalInstanceCommandFlags consolidates all command flags required to run a local instance in one struct
type RunLocalInstanceCommandFlags struct {
	Accel           bool
	Bridged         bool
	BridgeName      string
	BridgeIPAddress string
	Debug           bool
	Force           bool
	GDBPort         int
	MissingFiles    bool
	NoTrace         []string
	Ports           []string
	UDPPorts        []string
	SkipBuild       bool
	Memory          string
	Smp             int
	SyscallSummary  bool
	TapName         string
	Trace           bool
	Verbose         bool
	Arch            string
}

// MergeToConfig overrides configuration passed by argument with command flags values
func (flags *RunLocalInstanceCommandFlags) MergeToConfig(c *types.Config) error {
	c.Debugflags = []string{}

	if flags.Trace {
		c.Debugflags = []string{"trace", "debugsyscalls", "fault", "futex_trace"} // "futex_trace" is just for temporay compatibility
	}

	if flags.Debug {
		c.RunConfig.Debug = true

		c.Debugflags = append(c.Debugflags, "noaslr")

		if len(c.ProgramPath) > 0 {
			var elfFile *elf.File

			elfFile, err := api.GetElfFileInfo(c.ProgramPath)
			if err != nil {
				return err
			}

			if api.IsDynamicLinked(elfFile) {
				log.Errorf("Program %s must be linked statically", c.ProgramPath)
			}

			if !api.HasDebuggingSymbols(elfFile) {
				log.Errorf("Program %s must be compiled with debugging symbols", c.ProgramPath)
			}

		} else {
			log.Errorf("Debug executable not found (is this a package?)")
		}
	}

	if flags.SyscallSummary {
		c.Debugflags = append(c.Debugflags, "syscall_summary")
	}

	if (flags.Trace || flags.SyscallSummary) && !slices.Contains(c.Klibs, "strace") {
		c.Klibs = append(c.Klibs, "strace") // debugsyscalls, notrace, tracelist, syscall_summary
	}

	if flags.MissingFiles {
		c.Debugflags = append(c.Debugflags, "missing_files")
	}

	if flags.Memory != "" {
		c.RunConfig.Memory = flags.Memory
	}

	if flags.Smp > 1 {
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

	if flags.BridgeIPAddress != "" {
		c.RunConfig.BridgeIPAddress = flags.BridgeIPAddress
	}

	if len(flags.NoTrace) > 0 {
		c.NoTrace = flags.NoTrace
	}

	c.RunConfig.Verbose = flags.Verbose
	c.RunConfig.Bridged = flags.Bridged
	c.RunConfig.Accel = flags.Accel
	c.Force = flags.Force

	ports, err := PrepareNetworkPorts(flags.Ports)
	if err != nil {
		return err
	}

	for _, p := range ports {
		i, err := strconv.Atoi(p)
		if err == nil && i == flags.GDBPort {
			errstr := fmt.Sprintf("Port %d is forwarded and cannot be used as gdb port", flags.GDBPort)
			return errors.New(errstr)
		}

		portAlreadyExists := false
		for _, rconfigPort := range c.RunConfig.Ports {
			if rconfigPort == p {
				portAlreadyExists = true
				break
			}
		}

		if !portAlreadyExists {
			c.RunConfig.Ports = append(c.RunConfig.Ports, p)
		}

	}

	if len(flags.UDPPorts) != 0 {
		c.RunConfig.UDPPorts = append(c.RunConfig.UDPPorts, flags.UDPPorts...)
	}

	for _, port := range flags.Ports {
		conn, err := net.DialTimeout("tcp", ":"+port, time.Second)
		if err != nil {
			continue // assume port is not being used
		}

		if conn != nil {
			conn.Close()

			message := fmt.Sprintf("Port %v is being used by other application", port)
			lsofOut, _ := exec.Command("lsof", "-t", "-i", ":"+port).CombinedOutput()
			pid := strings.TrimSpace(string(lsofOut))
			if pid != "" {
				message += fmt.Sprintf(" (PID %s)", pid)
			}
			return errors.New(message)
		}
	}

	return nil
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

	flags.GDBPort, err = cmdFlags.GetInt("gdbport")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.MissingFiles, err = cmdFlags.GetBool("missing-files")
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

	flags.UDPPorts, err = cmdFlags.GetStringArray("udp")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.SkipBuild, err = cmdFlags.GetBool("skipbuild")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Memory, err = cmdFlags.GetString("memory")
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

	flags.BridgeIPAddress, err = cmdFlags.GetString("bridgeipaddress")
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
	cmdFlags.StringArrayP("udp", "", nil, "udp ports to forward")
	cmdFlags.BoolP("force", "f", false, "update images")
	cmdFlags.BoolP("debug", "d", false, "enable interactive debugger")
	cmdFlags.BoolP("trace", "", false, "enable required flags to trace")
	cmdFlags.IntP("gdbport", "g", 0, "qemu TCP port used for GDB interface")
	cmdFlags.StringArrayP("no-trace", "", nil, "do not trace syscall")
	cmdFlags.BoolP("verbose", "v", false, "verbose")
	cmdFlags.BoolP("bridged", "b", false, "bridge networking")
	cmdFlags.StringP("bridgename", "", "", "bridge name")
	cmdFlags.StringP("bridgeipaddress", "", "", "bridge ip address")
	cmdFlags.StringP("tapname", "t", "", "tap device name")
	cmdFlags.BoolP("skipbuild", "s", false, "skip building image")
	cmdFlags.Bool("accel", true, "use cpu virtualization extension")
	cmdFlags.StringP("memory", "m", "", "RAM size")
	cmdFlags.IntP("smp", "", 1, "number of threads to use")
	cmdFlags.Bool("syscall-summary", false, "print syscall summary on exit")
	cmdFlags.Bool("missing-files", false, "print list of files not found on image at exit")
}

// isIPAddressValid checks whether IP address is valid
func isIPAddressValid(ip string) bool {
	if net.ParseIP(ip) == nil {
		return false
	}

	return true
}

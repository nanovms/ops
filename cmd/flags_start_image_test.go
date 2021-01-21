package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateStartImageFlags(t *testing.T) {
	startImageFlags := newStartImageFlagSet("true")

	assert.Equal(t, startImageFlags.Ports, []string{"80,81,82-85"})
	assert.Equal(t, startImageFlags.Force, true)
	assert.Equal(t, startImageFlags.Debug, true)
	assert.Equal(t, startImageFlags.Trace, true)
	assert.Equal(t, startImageFlags.GDBPort, 1234)
	assert.Equal(t, startImageFlags.NoTrace, []string{"a"})
	assert.Equal(t, startImageFlags.Verbose, true)
	assert.Equal(t, startImageFlags.Bridged, true)

	assert.Equal(t, startImageFlags.TapName, "tap1")
	assert.Equal(t, startImageFlags.IPAddress, "192.168.0.1")
	assert.Equal(t, startImageFlags.Gateway, "192.168.1.254")
	assert.Equal(t, startImageFlags.Netmask, "255.255.0.0")
	assert.Equal(t, startImageFlags.SkipBuild, true)
	assert.Equal(t, startImageFlags.Accel, false)
	assert.Equal(t, startImageFlags.Smp, 2)
	assert.Equal(t, startImageFlags.SyscallSummary, true)

}

func TestStartImageFlagsMergeToConfig(t *testing.T) {
	startImageFlags := newStartImageFlagSet("false")
	startImageFlags.Debug = false

	config := &lepton.Config{}

	err := startImageFlags.MergeToConfig(config)

	assert.Nil(t, err, nil)

	expected := &lepton.Config{
		BuildDir:   "",
		Debugflags: []string{"trace", "debugsyscalls", "futex_trace", "fault", "syscall_summary"},
		Force:      true,
		NoTrace:    []string{"a"},
		RunConfig: lepton.RunConfig{
			Accel:   true,
			Bridged: true,
			CPUs:    2,
			Debug:   false,
			Gateway: "192.168.1.254",
			GdbPort: 1234,
			IPAddr:  "192.168.0.1",
			Mounts:  []string(nil),
			NetMask: "255.255.0.0",
			Ports:   []string{"80", "81", "82-85"},
			TapName: "tap1",
			Verbose: true,
		},
	}

	assert.Equal(t, expected, config)

}

func newStartImageFlagSet(debug string) *cmd.StartImageCommandFlags {
	flagSet := pflag.NewFlagSet("test", 0)

	cmd.PersistStartImageCommandFlags(flagSet)

	flagSet.Set("port", "80,81,82-85")
	flagSet.Set("force", "true")
	flagSet.Set("debug", debug)
	flagSet.Set("trace", "true")
	flagSet.Set("gdbport", "1234")
	flagSet.Set("no-trace", "a")
	flagSet.Set("verbose", "true")
	flagSet.Set("bridged", "true")
	flagSet.Set("tapname", "tap1")
	flagSet.Set("ip-address", "192.168.0.1")
	flagSet.Set("gateway", "192.168.1.254")
	flagSet.Set("netmask", "255.255.0.0")
	flagSet.Set("skipbuild", "true")
	flagSet.Set("manifest-name", "manifest.json")
	flagSet.Set("accel", "true")
	flagSet.Set("smp", "2")
	flagSet.Set("mounts", "files:/mnt/f")
	flagSet.Set("syscall-summary", "true")

	return cmd.NewStartImageCommandFlags(flagSet)
}

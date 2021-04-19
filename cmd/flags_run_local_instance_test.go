package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/types"

	"github.com/nanovms/ops/cmd"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateRunLocalInstanceFlags(t *testing.T) {
	runLocalInstanceFlags := newRunLocalInstanceFlagSet("true")

	assert.Equal(t, runLocalInstanceFlags.Ports, []string{"80,81,82-85"})
	assert.Equal(t, runLocalInstanceFlags.Force, true)
	assert.Equal(t, runLocalInstanceFlags.Debug, true)
	assert.Equal(t, runLocalInstanceFlags.Trace, true)
	assert.Equal(t, runLocalInstanceFlags.GDBPort, 1234)
	assert.Equal(t, runLocalInstanceFlags.NoTrace, []string{"a"})
	assert.Equal(t, runLocalInstanceFlags.Verbose, true)
	assert.Equal(t, runLocalInstanceFlags.Bridged, true)
	assert.Equal(t, runLocalInstanceFlags.BridgeName, "br1")

	assert.Equal(t, runLocalInstanceFlags.TapName, "tap1")
	assert.Equal(t, runLocalInstanceFlags.SkipBuild, true)
	assert.Equal(t, runLocalInstanceFlags.Accel, false)
	assert.Equal(t, runLocalInstanceFlags.Smp, 2)
	assert.Equal(t, runLocalInstanceFlags.SyscallSummary, true)

}

func TestRunLocalInstanceFlagsMergeToConfig(t *testing.T) {

	t.Run("should merge configuration", func(t *testing.T) {
		runLocalInstanceFlags := newRunLocalInstanceFlagSet("false")
		runLocalInstanceFlags.Debug = false

		c := &types.Config{}

		err := runLocalInstanceFlags.MergeToConfig(c)

		assert.Nil(t, err, nil)

		expected := &types.Config{
			BuildDir:   "",
			Debugflags: []string{"trace", "debugsyscalls", "futex_trace", "fault", "syscall_summary"},
			Force:      true,
			NoTrace:    []string{"a"},
			RunConfig: types.RunConfig{
				Accel:      true,
				Bridged:    true,
				BridgeName: "br1",
				CPUs:       2,
				Debug:      false,
				GdbPort:    1234,
				Mounts:     []string(nil),
				Ports:      []string{"80", "81", "82-85"},
				TapName:    "tap1",
				Verbose:    true,
			},
		}

		assert.Equal(t, expected, c)
	})

	t.Run("should join existing ports with flags ports de-duplicated", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		cmd.PersistRunLocalInstanceCommandFlags(flagSet)

		flagSet.Set("port", "80,81,82-85")

		runLocalInstanceFlags := cmd.NewRunLocalInstanceCommandFlags(flagSet)

		c := &types.Config{
			RunConfig: types.RunConfig{
				Ports: []string{"80", "90"},
			},
		}

		err := runLocalInstanceFlags.MergeToConfig(c)

		assert.Nil(t, err, nil)

		expected := &types.Config{
			Debugflags: []string{},
			RunConfig: types.RunConfig{
				Accel: true,
				CPUs:  1,
				Ports: []string{"80", "90", "81", "82-85"},
			},
		}

		assert.Equal(t, expected, c)
	})

}

func newRunLocalInstanceFlagSet(debug string) *cmd.RunLocalInstanceCommandFlags {
	flagSet := pflag.NewFlagSet("test", 0)

	cmd.PersistRunLocalInstanceCommandFlags(flagSet)

	flagSet.Set("port", "80,81,82-85")
	flagSet.Set("force", "true")
	flagSet.Set("debug", debug)
	flagSet.Set("trace", "true")
	flagSet.Set("gdbport", "1234")
	flagSet.Set("no-trace", "a")
	flagSet.Set("verbose", "true")
	flagSet.Set("bridged", "true")
	flagSet.Set("bridgename", "br1")
	flagSet.Set("tapname", "tap1")
	flagSet.Set("skipbuild", "true")
	flagSet.Set("manifest-name", "manifest.json")
	flagSet.Set("accel", "true")
	flagSet.Set("smp", "2")
	flagSet.Set("mounts", "files:/mnt/f")
	flagSet.Set("syscall-summary", "true")

	return cmd.NewRunLocalInstanceCommandFlags(flagSet)
}

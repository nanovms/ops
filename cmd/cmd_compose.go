package cmd

import (
	"fmt"
	"os"

	"github.com/nanovms/ops/qemu"
	"github.com/spf13/cobra"

	api "github.com/nanovms/ops/lepton"
)

// ComposeCommands provides support for running binary with nanos
func ComposeCommands() *cobra.Command {

	var cmdCompose = &cobra.Command{
		Use:       "compose",
		Short:     "orchestrate multiple unikernels",
		ValidArgs: []string{"up", "down"},
		Args:      cobra.OnlyValidArgs,
	}

	cmdCompose.AddCommand(composeUpCommand())
	cmdCompose.AddCommand(composeDownCommand())

	return cmdCompose
}

func composeDownCommand() *cobra.Command {
	var cmdDownCompose = &cobra.Command{
		Use:   "down",
		Short: "spin unikernels down",
		Run:   composeDownCommandHandler,
	}

	return cmdDownCompose
}

func composeUpCommand() *cobra.Command {
	var cmdUpCompose = &cobra.Command{
		Use:   "up",
		Short: "spin unikernels up",
		Run:   composeUpCommandHandler,
	}

	return cmdUpCompose
}

func composeDownCommandHandler(cmd *cobra.Command, args []string) {
	if qemu.OPSD == "" {
		fmt.Println("this command is only enabled if you have OPSD compiled in.")
		os.Exit(1)
	}

	c := api.NewConfig()

	p, ctx, err := getProviderAndContext(c, "onprem")
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	instances, err := p.GetInstances(ctx)
	if err != nil {
		fmt.Println(err)
	}

	for i := 0; i < len(instances); i++ {
		err = p.DeleteInstance(ctx, instances[i].Name)
		if err != nil {
			exitWithError(err.Error())
		}
	}
}

func composeUpCommandHandler(cmd *cobra.Command, args []string) {
	if qemu.OPSD == "" {
		fmt.Println("this command is only enabled if you have OPSD compiled in.")
		os.Exit(1)
	}

	flags := cmd.Flags()
	globalFlags := NewGlobalCommandFlags(flags)

	c := api.NewConfig()
	mergeContainer := NewMergeConfigContainer(globalFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	com := Compose{
		config: c,
	}
	com.UP()

}

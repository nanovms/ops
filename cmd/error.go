package cmd

import (
	"fmt"
	"os"

	"github.com/go-errors/errors"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func exitWithError(errs string) {
	fmt.Println(fmt.Sprintf(api.ErrorColor, errs))
	os.Exit(1)
}

func exitForCmd(cmd *cobra.Command, errs string) {
	fmt.Println(fmt.Sprintf(api.ErrorColor, errs))
	cmd.Help()
	os.Exit(1)
}

func panicOnError(err error) {
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
		panic(err)
	}
}

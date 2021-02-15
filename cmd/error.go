package cmd

import (
	"fmt"
	"os"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/constants"
	"github.com/spf13/cobra"
)

func exitWithError(errs string) {
	fmt.Println(fmt.Sprintf(constants.ErrorColor, errs))
	os.Exit(1)
}

func exitForCmd(cmd *cobra.Command, errs string) {
	fmt.Println(fmt.Sprintf(constants.ErrorColor, errs))
	cmd.Help()
	os.Exit(1)
}

func panicOnError(err error) {
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
		panic(err)
	}
}

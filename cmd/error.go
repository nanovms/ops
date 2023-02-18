package cmd

import (
	"os"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/log"
	"github.com/spf13/cobra"
)

func exitWithError(errs string) {
	log.Fatalf(errs)
}

func exitForCmd(cmd *cobra.Command, errs string) {
	log.Errorf(errs)
	cmd.Help()
	os.Exit(1)
}

func panicOnError(err error) {
	if err != nil {
		log.Errorf(err.(*errors.Error).ErrorStack())
		panic(err)
	}
}

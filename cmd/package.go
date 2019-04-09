package cmd

import (
	"os"
	"regexp"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	api "github.com/nanovms/ops/lepton"
)

func cmdListPackages(cmd *cobra.Command, args []string) {
	packages := api.GetPackageList()

	searchRegex, err := cmd.Flags().GetString("search")
	if err != nil {
		panic(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"PackageName", "Version", "Language", "Runtime", "Description"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor})

	table.SetRowLine(true)

	var r *regexp.Regexp
	var filter bool
	if len(searchRegex) > 0 {
		filter = true
		r, err = regexp.Compile(searchRegex)
		if err != nil {
			// If the regex cannot compile do not attempt to filter
			filter = false
		}
	}

	for key, val := range *packages {
		var row []string
		// If we are told to filter and get no matches then filter out the
		// current row. If we are not told to filter then just add the
		// row.
		if filter &&
			!(r.MatchString(val.Language) ||
				r.MatchString(val.Runtime) ||
				r.MatchString(key)) {
			continue
		}

		row = append(row, key)
		row = append(row, val.Version)
		row = append(row, val.Language)
		row = append(row, val.Runtime)
		row = append(row, val.Description)
		table.Append(row)
	}

	table.Render()
}

// PackageCommand gives package related commands
func PackageCommand() *cobra.Command {
	var search string
	var cmdPkgList = &cobra.Command{
		Use:   "list",
		Short: "list packages",
		Run:   cmdListPackages,
	}

	var cmdPkg = &cobra.Command{
		Use:       "package",
		Short:     "Package related commands",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"list"},
	}

	cmdPkgList.PersistentFlags().StringVarP(&search, "search", "s", "", "search package list")
	cmdPkg.AddCommand(cmdPkgList)
	return cmdPkg
}

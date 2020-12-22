package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	api "github.com/nanovms/ops/lepton"
)

func cmdListPackages(cmd *cobra.Command, args []string) {
	var packages *map[string]api.Package
	var err error
	local, _ := cmd.Flags().GetBool("local")

	if local {
		packages, err = api.GetLocalPackageList()
	} else {
		packages, err = api.GetPackageList()
	}
	if err != nil {
		log.Panicf("failed getting packages: %s", err)
	}

	searchRegex, err := cmd.Flags().GetString("search")
	if err != nil {
		panic(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"PackageName", "Version", "Language", "Runtime", "Description"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})

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

	// Sort the package list by packagename
	keys := make([]string, 0, len(*packages))
	for key := range *packages {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		var row []string
		// If we are told to filter and get no matches then filter out the
		// current row. If we are not told to filter then just add the
		// row.
		if filter &&
			!(r.MatchString((*packages)[key].Language) ||
				r.MatchString((*packages)[key].Runtime) ||
				r.MatchString(key)) {
			continue
		}

		row = append(row, key)
		row = append(row, (*packages)[key].Version)
		row = append(row, (*packages)[key].Language)
		row = append(row, (*packages)[key].Runtime)
		row = append(row, (*packages)[key].Description)
		table.Append(row)
	}

	table.Render()
}

func cmdGetPackage(cmd *cobra.Command, args []string) {
	_, err := api.DownloadPackage(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func cmdPackageDescribe(cmd *cobra.Command, args []string) {
	expackage := downloadAndExtractPackage(args[0])

	description := path.Join(expackage, "README.md")
	if _, err := os.Stat(description); err != nil {
		fmt.Println("Error: Package information not provided.")
		os.Exit(1)
	}

	file, err := os.Open(description)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer file.Close()

	fmt.Println("Information for " + args[0] + " package:")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func cmdPackageContents(cmd *cobra.Command, args []string) {
	expackage := downloadAndExtractPackage(args[0])

	filepath.Walk(expackage, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		contentpath := strings.Split(hostpath, expackage)[1]
		if contentpath == "" {
			return nil
		}
		if info.IsDir() {
			fmt.Println("Dir :" + contentpath)
		} else {
			fmt.Println("File :" + contentpath)
		}

		return nil
	})
}

// PackageCommands gives package related commands
func PackageCommands() *cobra.Command {
	var search string
	var cmdPkgList = &cobra.Command{
		Use:   "list",
		Short: "list packages",
		Run:   cmdListPackages,
	}

	var cmdGetPackage = &cobra.Command{
		Use:   "get [packagename]",
		Short: "download a package from ['ops pkg list'] to the local cache",
		Args:  cobra.MinimumNArgs(1),
		Run:   cmdGetPackage,
	}

	var cmdPackageDescribe = &cobra.Command{
		Use:   "describe [packagename]",
		Short: "display information of a package from ['ops pkg list']",
		Args:  cobra.MinimumNArgs(1),
		Run:   cmdPackageDescribe,
	}

	var cmdPackageContents = &cobra.Command{
		Use:   "contents [packagename]",
		Short: "list contents of a package from ['ops pkg list']",
		Args:  cobra.MinimumNArgs(1),
		Run:   cmdPackageContents,
	}
	var cmdPkg = &cobra.Command{
		Use:       "pkg",
		Short:     "Package related commands",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"list", "get", "describe", "contents"},
	}

	cmdPkgList.PersistentFlags().StringVarP(&search, "search", "s", "", "search package list")
	cmdPkgList.PersistentFlags().Bool("local", false, "display local packages")
	cmdPkg.AddCommand(cmdPkgList)
	cmdPkg.AddCommand(cmdGetPackage)
	cmdPkg.AddCommand(cmdPackageContents)
	cmdPkg.AddCommand(cmdPackageDescribe)
	return cmdPkg
}

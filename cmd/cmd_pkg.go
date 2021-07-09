package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
)

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

	var cmdAddPackage = &cobra.Command{
		Use:   "add [package]",
		Short: "push a folder or a .tar.gz archived package to the local cache",
		Args:  cobra.MinimumNArgs(1),
		Run:   cmdAddPackage,
	}

	var cmdFromDockerPackage = &cobra.Command{
		Use:   "from-docker [image]",
		Short: "create a package from an executable of a docker image",
		Args:  cobra.MinimumNArgs(1),
		Run:   cmdFromDockerPackage,
	}

	var cmdPkg = &cobra.Command{
		Use:       "pkg",
		Short:     "Package related commands",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"list", "get", "describe", "contents", "add", "load", "from-docker"},
	}

	cmdPkgList.PersistentFlags().StringVarP(&search, "search", "s", "", "search package list")
	cmdPkgList.PersistentFlags().Bool("local", false, "display local packages")

	cmdPackageContents.PersistentFlags().BoolP("local", "l", false, "local package")

	cmdAddPackage.PersistentFlags().StringP("name", "n", "", "name of the package")

	cmdFromDockerPackage.PersistentFlags().BoolP("quiet", "q", false, "quiet mode")
	cmdFromDockerPackage.PersistentFlags().Bool("verbose", false, "verbose mode")
	cmdFromDockerPackage.PersistentFlags().StringP("file", "f", "", "target executable")
	cmdFromDockerPackage.MarkPersistentFlagRequired("file")
	cmdFromDockerPackage.PersistentFlags().StringP("name", "n", "", "name of the package")

	cmdPkg.AddCommand(cmdPkgList)
	cmdPkg.AddCommand(cmdGetPackage)
	cmdPkg.AddCommand(cmdPackageContents)
	cmdPkg.AddCommand(cmdPackageDescribe)
	cmdPkg.AddCommand(cmdAddPackage)
	cmdPkg.AddCommand(cmdFromDockerPackage)
	cmdPkg.AddCommand(LoadCommand())
	return cmdPkg
}

func cmdListPackages(cmd *cobra.Command, args []string) {
	var packages *map[string]api.Package
	var err error
	local, _ := cmd.Flags().GetBool("local")

	if local {
		packages, err = api.GetLocalPackageList()
	} else {
		packages, err = api.GetPackageList(lepton.NewConfig())
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
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

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
	downloadPackage(args[0], lepton.NewConfig())
}

func cmdPackageDescribe(cmd *cobra.Command, args []string) {
	expackage := filepath.Join(packageDirectoryPath(), args[0])
	if _, err := os.Stat(expackage); os.IsNotExist(err) {
		expackage = downloadPackage(args[0], lepton.NewConfig())
	}

	description := path.Join(expackage, "README.md")
	if _, err := os.Stat(description); err != nil {
		log.Errorf("Error: Package information not provided.")
		return
	}

	file, err := os.Open(description)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	fmt.Println("Information for " + args[0] + " package:")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func cmdPackageContents(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	directoryPath := packageDirectoryPath()

	if local, _ := flags.GetBool("local"); local {
		directoryPath = localPackageDirectoryPath()
	}

	expackage := filepath.Join(directoryPath, args[0])
	if _, err := os.Stat(expackage); os.IsNotExist(err) {
		expackage = downloadPackage(args[0], lepton.NewConfig())
	}

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

func randomToken(n int) (string, error) {
	bytes := make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func cmdAddPackage(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	name, _ := flags.GetString("name")

	if name == "" {
		token, err := randomToken(8)
		if err != nil {
			log.Fatal(err)
		}

		name = token
	}

	extractFilePackage(args[0], name, lepton.NewConfig())

	fmt.Println(name)
}

func cmdFromDockerPackage(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	imageName := args[0]
	quiet, _ := flags.GetBool("quiet")
	verbose, _ := flags.GetBool("verbose")
	packageName, _ := flags.GetString("name")
	targetExecutable, _ := flags.GetString("file")

	packageName, _ = ExtractFromDockerImage(imageName, packageName, targetExecutable, quiet, verbose)
	fmt.Println(packageName)
}

// LoadCommand helps you to run application with package
func LoadCommand() *cobra.Command {
	var cmdLoadPackage = &cobra.Command{
		Use:   "load [packagename]",
		Short: "load and run a package from ['ops pkg list']",
		Args:  cobra.MinimumNArgs(1),
		Run:   loadCommandHandler,
	}

	persistentFlags := cmdLoadPackage.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistBuildImageCommandFlags(persistentFlags)
	PersistRunLocalInstanceCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	persistentFlags.BoolP("local", "l", false, "load local package")

	return cmdLoadPackage
}

func loadCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	runLocalInstanceFlags := NewRunLocalInstanceCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)
	pkgFlags.Package = args[0]

	c := lepton.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, buildImageFlags, runLocalInstanceFlags, pkgFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	packageFolder := filepath.Base(pkgFlags.PackagePath())
	executableName := c.Program
	if strings.Contains(executableName, packageFolder) {
		executableName = filepath.Base(executableName)
	} else {
		executableName = filepath.Join(lepton.PackageSysRootFolderName, executableName)
	}
	lepton.ValidateELF(filepath.Join(pkgFlags.PackagePath(), executableName))

	if !runLocalInstanceFlags.SkipBuild {
		if err = api.BuildImageFromPackage(pkgFlags.PackagePath(), *c); err != nil {
			panic(err)
		}
	}

	err = RunLocalInstance(c)
	if err != nil {
		exitWithError(err.Error())
	}
}

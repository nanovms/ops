package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-errors/errors"
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

	var cmdPkgSearch = &cobra.Command{
		Use:   "search [packagename]",
		Short: "search packages",
		Args:  cobra.ExactArgs(1),
		Run:   cmdSearchPackages,
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

	var cmdPkgLogin = &cobra.Command{
		Use:   "login [apikey]",
		Short: "login to pkghub account using the apikey",
		Args:  cobra.ExactArgs(1),
		Run:   cmdPkgLogin,
	}

	var cmdPkgPush = &cobra.Command{
		Use:   "push [local-package]",
		Short: "push the local package to packagehub",
		Args:  cobra.ExactArgs(1),
		Run:   cmdPkgPush,
	}

	var cmdPkg = &cobra.Command{
		Use:       "pkg",
		Short:     "Package related commands",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"list", "get", "describe", "contents", "add", "load", "from-docker", "login"},
	}

	cmdPkgList.PersistentFlags().StringVarP(&search, "search", "s", "", "search package list")
	cmdPkgList.PersistentFlags().Bool("local", false, "display local packages")

	cmdPackageContents.PersistentFlags().BoolP("local", "l", false, "local package")

	cmdAddPackage.PersistentFlags().StringP("name", "n", "", "name of the package")

	cmdFromDockerPackage.PersistentFlags().BoolP("quiet", "q", false, "quiet mode")
	cmdFromDockerPackage.PersistentFlags().Bool("verbose", false, "verbose mode")
	cmdFromDockerPackage.PersistentFlags().StringP("file", "f", "", "target executable")
	cmdFromDockerPackage.PersistentFlags().BoolP("copy", "c", false, "copy whole file system")
	cmdFromDockerPackage.MarkPersistentFlagRequired("file")
	cmdFromDockerPackage.PersistentFlags().StringP("name", "n", "", "name of the package")

	cmdPkg.AddCommand(cmdPkgList)
	cmdPkg.AddCommand(cmdPkgSearch)
	cmdPkg.AddCommand(cmdGetPackage)
	cmdPkg.AddCommand(cmdPackageContents)
	cmdPkg.AddCommand(cmdPackageDescribe)
	cmdPkg.AddCommand(cmdAddPackage)
	cmdPkg.AddCommand(cmdFromDockerPackage)
	cmdPkg.AddCommand(cmdPkgLogin)
	cmdPkg.AddCommand(cmdPkgPush)
	cmdPkg.AddCommand(LoadCommand())
	return cmdPkg
}

func cmdListPackages(cmd *cobra.Command, args []string) {
	var packages []api.Package
	var err error
	local, _ := cmd.Flags().GetBool("local")

	if local {
		packages, err = api.GetLocalPackageList()
		if err != nil {
			log.Errorf("failed getting packages: %s", err)
			return
		}
	} else {
		pkgList, err := api.GetPackageList(api.NewConfig())
		if err != nil {
			log.Errorf("failed getting packages: %s", err)
			return
		}
		packages = pkgList.List()
	}

	searchRegex, err := cmd.Flags().GetString("search")
	if err != nil {
		panic(err)
	}

	table := pkgTable(packages)

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
	var rows [][]string
	for _, pkg := range packages {
		var row []string
		// If we are told to filter and get no matches then filter out the
		// current row. If we are not told to filter then just add the
		// row.
		if filter &&
			!(r.MatchString(pkg.Language) ||
				r.MatchString(pkg.Runtime) ||
				r.MatchString(pkg.Name) ||
				r.MatchString(pkg.Namespace)) {
			continue
		}
		row = append(row, pkg.Namespace)
		row = append(row, pkg.Name)
		row = append(row, pkg.Version)
		row = append(row, pkg.Language)
		row = append(row, pkg.Runtime)
		row = append(row, pkg.Description)
		rows = append(rows, row)
	}

	for _, row := range rows {
		table.Append(row)
	}

	table.Render()
}

func cmdSearchPackages(cmd *cobra.Command, args []string) {
	q := args[0]

	pkgs, err := api.SearchPackages(q)
	if err != nil {
		log.Errorf("Error while searching packages: %s", err.Error())
		return
	}
	if len(pkgs.Packages) == 0 {
		fmt.Println("No packages found.")
		return
	}
	table := pkgTable(pkgs.Packages)

	var rows [][]string
	for _, pkg := range pkgs.Packages {
		var row []string

		row = append(row, pkg.Namespace)
		row = append(row, pkg.Name)
		row = append(row, pkg.Version)
		row = append(row, pkg.Language)
		row = append(row, pkg.Runtime)
		row = append(row, pkg.Description)
		rows = append(rows, row)
	}

	for _, row := range rows {
		table.Append(row)
	}

	table.Render()
}

func cmdGetPackage(cmd *cobra.Command, args []string) {
	identifier := args[0]
	tokens := strings.Split(identifier, "/")
	if len(tokens) < 2 {
		log.Fatal(errors.New("invalid package name. expected format <namespace>/<pkg>:<version>"))
	}
	downloadPackage(args[0], api.NewConfig())
}

func cmdPackageDescribe(cmd *cobra.Command, args []string) {
	identifier := args[0]
	tokens := strings.Split(identifier, "/")
	if len(tokens) < 2 {
		log.Fatal(errors.New("invalid package name. expected format <namespace>/<pkg>:<version>"))
	}
	expackage := filepath.Join(packageDirectoryPath(), args[0])
	if _, err := os.Stat(expackage); os.IsNotExist(err) {
		expackage = downloadPackage(args[0], api.NewConfig())
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
	identifier := args[0]
	tokens := strings.Split(identifier, "/")
	if len(tokens) < 2 {
		log.Fatal(errors.New("invalid package name. expected format <namespace>/<pkg>:<version>"))
	}

	directoryPath := packageDirectoryPath()

	if local, _ := flags.GetBool("local"); local {
		directoryPath = localPackageDirectoryPath()
	}

	expackage := filepath.Join(directoryPath, strings.ReplaceAll(args[0], ":", "_"))
	if _, err := os.Stat(expackage); os.IsNotExist(err) {
		expackage = downloadPackage(args[0], api.NewConfig())
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

	extractFilePackage(args[0], name, api.NewConfig())

	fmt.Println(name)
}

func cmdFromDockerPackage(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	imageName := args[0]
	quiet, _ := flags.GetBool("quiet")
	verbose, _ := flags.GetBool("verbose")
	packageName, _ := flags.GetString("name")
	targetExecutable, _ := flags.GetString("file")
	copyWholeFS, _ := flags.GetBool("copy")

	packageName, _ = ExtractFromDockerImage(imageName, packageName, targetExecutable, quiet, verbose, copyWholeFS)
	fmt.Println(packageName)
}

func cmdPkgLogin(cmd *cobra.Command, args []string) {
	apikey := args[0]
	resp, err := api.ValidateAPIKey(apikey)
	if err != nil {
		log.Fatal(err)
	}
	err = api.StoreCredentials(api.Credentials{
		Username: resp.Username,
		APIKey:   apikey,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Login Successful as user %s\n", resp.Username)
}

func cmdPkgPush(cmd *cobra.Command, args []string) {
	packageFolder := args[0]
	creds, err := api.ReadCredsFromLocal()
	if err != nil {
		if err == api.ErrCredentialsNotExist {
			// for a better error message
			log.Fatal(errors.New("user is not logged in. use 'ops pkg login' first"))
		} else {
			log.Fatal(err)
		}
	}
	name, version := api.GetPkgnameAndVersion(packageFolder)
	pkgList, err := api.GetLocalPackageList()
	if err != nil {
		log.Fatal(err)
	}
	localPackages := filepath.Join(api.GetOpsHome(), "local_packages")
	var foundPkg api.Package
	for _, pkg := range pkgList {
		if pkg.Name == name && pkg.Version == version {
			foundPkg = pkg
			break
		}
	}
	if foundPkg.Name == "" {
		log.Fatalf("no local package with the name %s found", packageFolder)
	}

	// build the archive here
	archiveName := filepath.Join(localPackages, packageFolder) + ".tar.gz"
	err = lepton.CreateTarGz(filepath.Join(localPackages, packageFolder), archiveName)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(archiveName)

	req, err := lepton.BuildRequestForArchiveUpload(name, foundPkg, archiveName)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set(lepton.APIKeyHeader, creds.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	// if the package is uploaded successfully then pkghub redirects to home page
	if resp.StatusCode != http.StatusOK {
		log.Fatal(errors.New("there was as an error while uploading the archive"))
	} else {
		fmt.Println("Package was uploaded successfully.")
	}

}

func randomToken(n int) (string, error) {
	bytes := make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
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
	PersistNanosVersionCommandFlags(persistentFlags)
	persistentFlags.BoolP("local", "l", false, "load local package")

	return cmdLoadPackage
}

func loadCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	runLocalInstanceFlags := NewRunLocalInstanceCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)
	pkgFlags.Package = args[0]

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, buildImageFlags, runLocalInstanceFlags, pkgFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	packageFolder := filepath.Base(pkgFlags.PackagePath())
	executableName := c.Program
	if strings.Contains(executableName, packageFolder) {
		executableName = filepath.Base(executableName)
	} else {
		executableName = filepath.Join(api.PackageSysRootFolderName, executableName)
	}
	api.ValidateELF(filepath.Join(pkgFlags.PackagePath(), executableName))

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

func pkgTable(packages []api.Package) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Namespace", "PackageName", "Version", "Language", "Runtime", "Description"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})

	table.SetRowLine(true)

	// Sort the package list by packagename
	keys := make([]string, 0, len(packages))
	for _, pkg := range packages {
		keys = append(keys, pkg.Name)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	return table
}

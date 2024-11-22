package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"runtime"

	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/provider/onprem"
	"github.com/nanovms/ops/types"

	"github.com/go-errors/errors"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// PackageCommands gives package related commands
func PackageCommands() *cobra.Command {

	var cmdPkgSearch = &cobra.Command{
		Use:   "search [packagename]",
		Short: "search packages",
		Args:  cobra.ExactArgs(1),
		Run:   cmdSearchPackages,
	}

	var cmdPkgLogin = &cobra.Command{
		Use:   "login [apikey]",
		Short: "login to pkghub account using the apikey",
		Args:  cobra.ExactArgs(1),
		Run:   cmdPkgLogin,
	}

	var cmdPkg = &cobra.Command{
		Use:       "pkg",
		Short:     "Package related commands",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"list", "get", "describe", "delete", "contents", "add", "load", "from-docker", "login", "from-pkg"},
	}

	cmdPkgSearch.PersistentFlags().StringP("arch", "", "", "set different architecture")

	cmdPkg.AddCommand(addCommand())
	cmdPkg.AddCommand(getCommand())
	cmdPkg.AddCommand(contentsCommand())
	cmdPkg.AddCommand(describeCommand())
	cmdPkg.AddCommand(fromDockerCommand())
	cmdPkg.AddCommand(fromRunCommand())
	cmdPkg.AddCommand(fromPackageCommand())
	cmdPkg.AddCommand(listCommand())
	cmdPkg.AddCommand(LoadCommand())
	cmdPkg.AddCommand(DeleteCommand())
	cmdPkg.AddCommand(pushCommand())

	cmdPkg.AddCommand(cmdPkgSearch)
	cmdPkg.AddCommand(cmdPkgLogin)
	return cmdPkg
}

func getPkgArch() string {
	rt := runtime.GOARCH
	if rt == "amd64" {
		return "x86_64"
	}

	return rt
}

// translate amd64 -> x86_64
func getPreferredArch() string {

	parch := "amd64"
	if api.AltGOARCH != "" {
		if api.AltGOARCH == "arm64" {
			parch = "arm64"
		}
	} else {
		if api.RealGOARCH == "arm64" {
			parch = "arm64"
		}
	}

	if parch == "amd64" {
		return "x86_64"
	}

	return parch
}

func cmdListPackages(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, pkgFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	var packages []api.Package
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
		fmt.Printf(err.Error())
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		fmt.Printf(err.Error())
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

	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(packages)
		return
	}

	rt := getPreferredArch()

	var rows [][]string
	for _, pkg := range packages {
		var row []string
		// If we are told to filter and get no matches then filter out the
		// current row. If we are not told to filter then just add the
		// row.
		if pkg.Arch != "" && pkg.Arch != rt {
			continue
		}

		if filter &&
			!(r.MatchString(pkg.Language) ||
				r.MatchString(pkg.Name) ||
				r.MatchString(pkg.Namespace)) {
			continue
		}
		row = append(row, pkg.Namespace)
		row = append(row, pkg.Name)
		row = append(row, pkg.Version)
		row = append(row, pkg.Language)
		row = append(row, pkg.Arch)
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

	arch, err := cmd.Flags().GetString("arch")
	if err != nil {
		exitWithError(err.Error())
	}

	if arch != "" {
		if arch != "arm64" && arch != "amd64" {
			exitWithError("unknown architecture")
		}
	}

	var pkgs *api.PackageList

	rt := getPkgArch()

	if arch != "" {
		pkgs, err = api.SearchPackagesWithArch(q, arch)
		if err != nil {
			log.Errorf("Error while searching packages: %s", err.Error())
			return
		}
	} else {
		pkgs, err = api.SearchPackagesWithArch(q, rt)
		if err != nil {
			log.Errorf("Error while searching packages: %s", err.Error())
			return
		}
	}

	if len(pkgs.Packages) == 0 {
		fmt.Println("No packages found.")
		return
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		fmt.Printf(err.Error())
	}

	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(pkgs.Packages)
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
		row = append(row, pkg.Arch)
		row = append(row, pkg.Description)
		rows = append(rows, row)
	}

	for _, row := range rows {
		table.Append(row)
	}

	table.Render()
}

func getCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	identifier := args[0]
	tokens := strings.Split(identifier, "/")
	if len(tokens) < 2 {
		log.Fatal(errors.New("invalid package name. expected format <namespace>/<pkg>:<version>"))
	}

	downloadPackage(args[0], c)
}

func describeCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, pkgFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	identifier := args[0]
	tokens := strings.Split(identifier, "/")
	if len(tokens) < 2 {
		log.Fatal(errors.New("invalid package name. expected format <namespace>/<pkg>:<version>"))
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		fmt.Printf(err.Error())
	}

	pkgFlags.Package = args[0]

	expackage := pkgFlags.PackagePath()

	if _, err := os.Stat(expackage); os.IsNotExist(err) {
		expackage, err = downloadPackage(args[0], api.NewConfig())
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
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

	if jsonOutput {
		lines := ""

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines += scanner.Text() + "\n"
		}

		if err := scanner.Err(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		json.NewEncoder(os.Stdout).Encode(lines)
		return
	}

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
	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, pkgFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	local, _ := cmd.Flags().GetBool("local")

	if !local {
		identifier := args[0]
		tokens := strings.Split(identifier, "/")
		if len(tokens) < 2 {
			log.Fatal(errors.New("invalid package name. expected format <namespace>/<pkg>:<version>"))
		}
	}

	pkgFlags.Package = args[0]

	expackage := pkgFlags.PackagePath()
	if _, err := os.Stat(expackage); os.IsNotExist(err) {
		expackage, err = downloadPackage(args[0], api.NewConfig())
		if err != nil {
			fmt.Println(err)
		}
	}

	jsonOutput, err := flags.GetBool("json")
	if err != nil {
		fmt.Printf(err.Error())
	}

	files := []fdr{}

	filepath.Walk(expackage, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		contentpath := strings.Split(hostpath, expackage)[1]
		if contentpath == "" {
			return nil
		}
		if info.IsDir() {
			files = append(files, fdr{Name: contentpath, Dir: true})
		} else {
			files = append(files, fdr{Name: contentpath, Dir: false})
		}

		return nil
	})

	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(files)
	} else {
		for i := 0; i < len(files); i++ {
			if files[i].Dir {
				fmt.Println("Dir :" + files[i].Name)
			} else {
				fmt.Println("File :" + files[i].Name)
			}
		}
	}
}

type fdr struct {
	Name string
	Dir  bool
}

func addCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	c := api.NewConfig()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, pkgFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	name, _ := flags.GetString("name")

	if name == "" {
		token, err := randomToken(8)
		if err != nil {
			log.Fatal(err)
		}

		name = token
	}

	extractFilePackage(args[0], name, pkgFlags.Parch(), api.NewConfig())

	fmt.Println(name)
}

func fromDockerCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	c := api.NewConfig()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, pkgFlags, nanosVersionFlags, buildImageFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	imageName := args[0]
	quiet, _ := flags.GetBool("quiet")
	verbose, _ := flags.GetBool("verbose")
	packageName, _ := flags.GetString("name")
	targetExecutable, _ := flags.GetString("file")
	copyWholeFS, _ := flags.GetBool("copy")

	cmdArgs, err := flags.GetStringArray("args")
	if err != nil {
		exitWithError(err.Error())
	}

	packageName, _ = ExtractFromDockerImage(imageName, packageName, pkgFlags.Parch(), targetExecutable, quiet, verbose, copyWholeFS, cmdArgs)
	fmt.Println(packageName)
}

func fromPackageCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	c := api.NewConfig()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, pkgFlags, nanosVersionFlags, buildImageFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	newpkg, _ := flags.GetString("name")
	if newpkg == "" {
		fmt.Println("missing new pkg name")
		os.Exit(1)
	}

	version, _ := flags.GetString("version")
	if version == "" {
		fmt.Println("missing new version")
		os.Exit(1)
	}

	oldpkg := args[0]
	oldpkg = strings.ReplaceAll(oldpkg, ":", "_")

	o := path.Join(api.GetOpsHome(), "packages", pkgFlags.Parch(), oldpkg)
	ppath := o + "/package.manifest"
	oldConfig := &types.Config{}
	unWarpConfig(ppath, oldConfig)

	api.ClonePackage(oldpkg, newpkg, version, pkgFlags.Parch(), oldConfig, c)
}

func fromRunCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	c := api.NewConfig()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, pkgFlags, nanosVersionFlags, buildImageFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	newpkg, _ := flags.GetString("name")
	if newpkg == "" {
		fmt.Println("missing new pkg name")
		os.Exit(1)
	}

	version, _ := flags.GetString("version")
	if version == "" {
		fmt.Println("missing new version")
		os.Exit(1)
	}

	program := args[0]
	c.Program = program
	c.ProgramPath, err = filepath.Abs(c.Program)
	if err != nil {
		exitWithError(err.Error())
	}
	checkProgramExists(c.Program)

	api.CreatePackageFromRun(newpkg, version, pkgFlags.Parch(), c)
}

func pushCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	c := api.NewConfig()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, pkgFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	pkgIdentifier := args[0]
	creds, err := api.ReadCredsFromLocal()
	if err != nil {
		if err == api.ErrCredentialsNotExist {
			// for a better error message
			log.Fatal(errors.New("user is not logged in. use 'ops pkg login' first"))
		} else {
			log.Fatal(err)
		}
	}

	private, _ := flags.GetBool("private")
	ns, name, version := api.GetNSPkgnameAndVersion(pkgIdentifier)
	pkgList, err := api.GetLocalPackageList()
	if err != nil {
		log.Fatal(err)
	}

	_, packageFolder := api.ExtractNS(pkgIdentifier)
	localPackages := api.LocalPackagesRoot
	var foundPkg api.Package

	for _, pkg := range pkgList {
		// trip the "v" if provided in the version
		if pkg.Name == name && strings.TrimPrefix(pkg.Version, "v") == strings.TrimPrefix(version, "v") {
			foundPkg = pkg
			break
		}
	}
	if foundPkg.Name == "" {
		log.Fatalf("no local package with the name %s found", packageFolder)
	}

	// build the archive here
	archiveName := filepath.Join(localPackages, pkgFlags.Parch(), packageFolder) + ".tar.gz"

	err = api.CreateTarGz(filepath.Join(localPackages, pkgFlags.Parch(), packageFolder), archiveName)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(archiveName)

	req, err := api.BuildRequestForArchiveUpload(ns, name, foundPkg, archiveName, private, pkgFlags.Parch())
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set(api.APIKeyHeader, creds.APIKey)
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

func randomToken(n int) (string, error) {
	bytes := make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// DeleteCommand helps you to run application with package
func DeleteCommand() *cobra.Command {
	var cmdLoadPackage = &cobra.Command{
		Use:   "delete [packagename]",
		Short: "delete a package",
		Args:  cobra.MinimumNArgs(1),
		Run:   deleteCommandHandler,
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

func addCommand() *cobra.Command {

	var cmdAddPackage = &cobra.Command{
		Use:   "add [package]",
		Short: "push a folder or a .tar.gz archived package to the local cache",
		Args:  cobra.MinimumNArgs(1),
		Run:   addCommandHandler,
	}
	persistentFlags := cmdAddPackage.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)

	persistentFlags.StringP("name", "", "", "name of the package")
	persistentFlags.BoolP("local", "l", false, "load local package")

	return cmdAddPackage
}

func getCommand() *cobra.Command {
	var cmdGetPackage = &cobra.Command{
		Use:   "get [packagename]",
		Short: "download a package from ['ops pkg list'] to the local cache",
		Args:  cobra.MinimumNArgs(1),
		Run:   getCommandHandler,
	}

	persistentFlags := cmdGetPackage.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)

	return cmdGetPackage
}

func contentsCommand() *cobra.Command {
	var cmdContentsPackage = &cobra.Command{
		Use:   "contents [packagename]",
		Short: "list contents of a package from ['ops pkg list']",
		Args:  cobra.MinimumNArgs(1),
		Run:   cmdPackageContents,
	}

	persistentFlags := cmdContentsPackage.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	persistentFlags.BoolP("local", "l", false, "load local package")

	return cmdContentsPackage
}

func describeCommand() *cobra.Command {
	var cmdDescribePackage = &cobra.Command{
		Use:   "describe [packagename]",
		Short: "describe a package from ['ops pkg list']",
		Args:  cobra.MinimumNArgs(1),
		Run:   describeCommandHandler,
	}

	persistentFlags := cmdDescribePackage.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	persistentFlags.BoolP("local", "l", false, "load local package")

	return cmdDescribePackage
}

func fromDockerCommand() *cobra.Command {

	var cmdFromDocker = &cobra.Command{
		Use:   "from-docker [image]",
		Short: "create a package from an executable of a docker image",
		Args:  cobra.MinimumNArgs(1),
		Run:   fromDockerCommandHandler,
	}

	persistentFlags := cmdFromDocker.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistBuildImageCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)

	persistentFlags.BoolP("quiet", "q", false, "quiet mode")
	persistentFlags.Bool("verbose", false, "verbose mode")
	persistentFlags.StringP("file", "", "", "target executable")
	persistentFlags.BoolP("copy", "", false, "copy whole file system")
	persistentFlags.StringP("name", "", "", "name of the package")
	persistentFlags.BoolP("local", "l", false, "load local package")

	return cmdFromDocker
}

func fromPackageCommand() *cobra.Command {
	var cmdFromPackage = &cobra.Command{
		Use:   "from-pkg [old]",
		Short: "create a new package from an existing package",
		Run:   fromPackageCommandHandler,
	}

	persistentFlags := cmdFromPackage.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistBuildImageCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)

	persistentFlags.StringP("name", "", "", "name of the new package")
	persistentFlags.StringP("version", "", "", "version of the package")
	persistentFlags.BoolP("local", "l", false, "load local package")

	return cmdFromPackage
}

func fromRunCommand() *cobra.Command {
	var cmdFromRunPackage = &cobra.Command{
		Use:   "from-run [binary]",
		Short: "create a new package from an existing binary",
		Run:   fromRunCommandHandler,
	}

	persistentFlags := cmdFromRunPackage.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistBuildImageCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)

	persistentFlags.BoolP("local", "l", false, "load local package")
	persistentFlags.StringP("name", "", "", "name of the new package")
	persistentFlags.StringP("version", "", "", "version of the package")

	return cmdFromRunPackage
}

func listCommand() *cobra.Command {
	var cmdListPackage = &cobra.Command{
		Use:   "list",
		Short: "list packages",
		Run:   cmdListPackages,
	}

	var search string

	persistentFlags := cmdListPackage.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	persistentFlags.BoolP("local", "l", false, "load local package")
	persistentFlags.StringVarP(&search, "search", "s", "", "search package list")

	return cmdListPackage
}

func pushCommand() *cobra.Command {
	var cmdPushPackage = &cobra.Command{
		Use:   "push [local-package]",
		Short: "push the local package to packagehub",
		Args:  cobra.ExactArgs(1),
		Run:   pushCommandHandler,
	}

	persistentFlags := cmdPushPackage.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	persistentFlags.BoolP("local", "l", false, "load local package")
	persistentFlags.BoolP("private", "p", false, "set the package as private")

	return cmdPushPackage
}

func deleteCommandHandler(cmd *cobra.Command, args []string) {
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

	local, _ := cmd.Flags().GetBool("local")

	if local {
		err := os.RemoveAll(pkgFlags.PackagePath())
		if err != nil {
			fmt.Println(err)
		}

	} else {
		fmt.Println("not implemented for public pkgs atm - please use the website instead")
		os.Exit(1)
	}
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

	if c.Mounts != nil {
		err = onprem.AddVirtfsShares(c)
		if err != nil {
			exitWithError("Failed to add VirtFS shares: " + err.Error())
		}
	}

	if !runLocalInstanceFlags.SkipBuild {
		if err = api.BuildImageFromPackage(pkgFlags.PackagePath(), *c); err != nil {
			exitWithError("Failed to build image from package, error is: " + err.Error())
		}
	}

	err = RunLocalInstance(c)
	if err != nil {
		exitWithError(err.Error())
	}
}

func pkgTable(packages []api.Package) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Namespace", "PackageName", "Version", "Language", "CPU Arch", "Description"})
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

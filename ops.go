package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	api "github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func panicOnError(err error) {
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
		panic(err)
	}
}

func unWarpConfig(file string) *api.Config {
	var c api.Config
	if file != "" {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(data, &c)
		if err != nil {
			panic(err)
		}
	}
	return &c
}

func runCommandHandler(cmd *cobra.Command, args []string) {
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}

	force, err := strconv.ParseBool(cmd.Flag("force").Value.String())
	if err != nil {
		panic(err)
	}

	nightly, err := strconv.ParseBool(cmd.Flag("nightly").Value.String())
	if err != nil {
		panic(err)
	}

	debugflags, err := strconv.ParseBool(cmd.Flag("debug").Value.String())
	if err != nil {
		panic(err)
	}

	verbose, err := strconv.ParseBool(cmd.Flag("verbose").Value.String())
	if err != nil {
		panic(err)
	}

	bridged, err := strconv.ParseBool(cmd.Flag("bridged").Value.String())
	if err != nil {
		panic(err)
	}

	targetRoot, err := cmd.Flags().GetString("target-root")
	if err != nil {
		panic(err)
	}

	tapDeviceName, _ := cmd.Flags().GetString("tapname")
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	cmdargs, _ := cmd.Flags().GetStringArray("args")
	c := unWarpConfig(config)
	c.Args = append(c.Args, cmdargs...)
	c.Program = args[0]
	if debugflags {
		c.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}
	c.TargetRoot = targetRoot

	c.RunConfig.TapName = tapDeviceName
	c.RunConfig.Verbose = verbose
	c.RunConfig.Bridged = bridged
	c.NightlyBuild = nightly
	c.Force = force

	buildImages(c)

	ports := []int{}
	port, err := cmd.Flags().GetStringArray("port")

	if err != nil {
		panic(err)
	}
	for _, p := range port {
		i, err := strconv.Atoi(p)
		if err != nil {
			panic(err)
		}
		ports = append(ports, i)
	}

	fmt.Printf("booting %s ...\n", api.FinalImg)

	InitDefaultRunConfigs(c, ports)
	hypervisor.Start(&c.RunConfig)
}

func buildImages(c *api.Config) {
	var err error
	if c.NightlyBuild {
		err = api.DownloadImages(api.DevBaseUrl, c.Force)
	} else {
		err = api.DownloadImages(api.ReleaseBaseUrl, c.Force)
	}
	panicOnError(err)
	err = api.BuildImage(*c)
	panicOnError(err)
}

func buildCommandHandler(cmd *cobra.Command, args []string) {
	targetRoot, _ := cmd.Flags().GetString("target-root")
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	c.Program = args[0]
	c.TargetRoot = targetRoot
	buildImages(c)
	fmt.Printf("Bootable image file:%s\n", api.FinalImg)
}

func printManifestHandler(cmd *cobra.Command, args []string) {
	targetRoot, _ := cmd.Flags().GetString("target-root")
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	c.Program = args[0]
	c.TargetRoot = targetRoot
	m, err := api.BuildManifest(c)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(m.String())
}

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Println(api.Version)
}

func updateCommandHandler(cmd *cobra.Command, args []string) {
	fmt.Println("Checking for updates...")
	err := api.DoUpdate(fmt.Sprintf(api.OpsReleaseUrl, runtime.GOOS))
	if err != nil {
		fmt.Println("Failed to update.", err)
	} else {
		fmt.Println("Successfully updated ops. Please restart.")
	}
	os.Exit(0)
}

func runningAsRoot() bool {
	cmd := exec.Command("id", "-u")
	output, _ := cmd.Output()
	i, _ := strconv.Atoi(string(output[:len(output)-1]))
	return i == 0
}

func netSetupCommandHandler(cmd *cobra.Command, args []string) {
	if !runningAsRoot() {
		fmt.Println("net command needs root permission")
		return
	}
	if err := setupBridgeNetwork(); err != nil {
		panic(err)
	}
}

func netResetCommandHandler(cmd *cobra.Command, args []string) {
	if !runningAsRoot() {
		fmt.Println("net command needs root permission")
		return
	}
	if err := resetBridgeNetwork(); err != nil {
		panic(err)
	}
}

// merge userconfig to package config, user config takes precedence
func mergeConfigs(pkgConfig *api.Config, usrConfig *api.Config) *api.Config {

	pkgConfig.Args = append(pkgConfig.Args, usrConfig.Args...)
	pkgConfig.Dirs = append(pkgConfig.Dirs, usrConfig.Dirs...)
	pkgConfig.Files = append(pkgConfig.Files, usrConfig.Files...)

	if pkgConfig.MapDirs == nil {
		pkgConfig.MapDirs = make(map[string]string)
	}

	if pkgConfig.Env == nil {
		pkgConfig.Env = make(map[string]string)
	}

	for k, v := range usrConfig.MapDirs {
		pkgConfig.MapDirs[k] = v
	}

	for k, v := range usrConfig.Env {
		pkgConfig.Env[k] = v
	}

	if usrConfig.RunConfig.Imagename != "" {
		pkgConfig.RunConfig.Imagename = usrConfig.RunConfig.Imagename
	}

	if usrConfig.RunConfig.Memory != "" {
		pkgConfig.RunConfig.Memory = usrConfig.RunConfig.Memory
	}
	return pkgConfig
}

func loadCommandHandler(cmd *cobra.Command, args []string) {
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}

	// Get current directory
	currentPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Create temp directory to extract page
	tempPath, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		panic(err)
	}

	localpackage := api.DownloadPackage(args[0])

	fmt.Printf("Extracting %s...\n", localpackage)
	api.ExtractPackage(localpackage, tempPath)

	// load the package manifest
	manifest := filepath.Join(tempPath, args[0], "package.manifest")
	if _, err := os.Stat(manifest); err != nil {
		panic(err)
	}

	pkgConfig := unWarpConfig(manifest)

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)

	// Get temp directory path
	if c.LocalTMPDir == "" {
		tempDirectoryPath, err := GetTMPPathByProgramPath(filepath.Join(currentPath, pkgConfig.Program))
		if err != nil {
			panic(err)
		}
		c.LocalTMPDir = tempDirectoryPath
	}

	// Remove the folder first.
	_ = os.RemoveAll(path.Join(c.LocalTMPDir, args[0]))

	// Move package from tempPath to tempDirectoryPath and remove tempPath
	if err := os.Rename(filepath.Join(tempPath, args[0]), filepath.Join(c.LocalTMPDir, args[0])); err != nil {
		panic(err)
	}
	_ = os.RemoveAll(tempPath)

	debugflags, err := strconv.ParseBool(cmd.Flag("debug").Value.String())
	if err != nil {
		panic(err)
	}

	verbose, err := strconv.ParseBool(cmd.Flag("verbose").Value.String())
	if err != nil {
		panic(err)
	}

	bridged, err := strconv.ParseBool(cmd.Flag("bridged").Value.String())
	if err != nil {
		panic(err)
	}

	force, err := strconv.ParseBool(cmd.Flag("force").Value.String())
	if err != nil {
		panic(err)
	}

	nightly, err := strconv.ParseBool(cmd.Flag("nightly").Value.String())
	if err != nil {
		panic(err)
	}

	cmdargs, _ := cmd.Flags().GetStringArray("args")
	c.Args = append(c.Args, cmdargs...)

	if debugflags {
		pkgConfig.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}

	c = mergeConfigs(pkgConfig, c)
	pkgConfig.RunConfig.Verbose = verbose
	pkgConfig.RunConfig.Bridged = bridged
	pkgConfig.NightlyBuild = nightly
	pkgConfig.Force = force

	if err = api.BuildFromPackage(path.Join(c.LocalTMPDir, args[0]), c); err != nil {
		panic(err)
	}

	ports := []int{}
	port, err := cmd.Flags().GetStringArray("port")

	if err != nil {
		panic(err)
	}
	for _, p := range port {
		i, err := strconv.Atoi(p)
		if err != nil {
			panic(err)
		}
		ports = append(ports, i)
	}

	fmt.Printf("booting %s ...\n", api.FinalImg)

	InitDefaultRunConfigs(c, ports)
	hypervisor.Start(&c.RunConfig)

}

func InitDefaultRunConfigs(c *api.Config, ports []int) {

	// Get full path to program
	programAbsPath, err := filepath.Abs(c.Program)
	if err != nil {
		panic(err)
	}

	// Get temp directory path
	if c.LocalTMPDir == "" {
		tempDirectoryPath, err := GetTMPPathByProgramPath(programAbsPath)
		if err != nil {
			panic(err)
		}
		c.LocalTMPDir = tempDirectoryPath
	}

	if c.RunConfig.Imagename == "" {
		c.RunConfig.Imagename = filepath.Join(c.LocalTMPDir, api.FinalImg)
	}
	if c.RunConfig.Memory == "" {
		c.RunConfig.Memory = "2G"
	}
	c.RunConfig.Ports = append(c.RunConfig.Ports, ports...)
}

func packageManifestChanged(fino os.FileInfo, remoteUrl string) bool {
	res, err := http.Head(remoteUrl)
	if err != nil {
		panic(err)
	}
	return fino.Size() != res.ContentLength
}

func cmdListPackages(cmd *cobra.Command, args []string) {

	var err error
	packageManifest := api.GetPackageManifestFile()
	stat, err := os.Stat(packageManifest)
	if os.IsNotExist(err) || packageManifestChanged(stat, api.PackageManifestURL) {
		if err = api.DownloadFile(packageManifest, api.PackageManifestURL, 10); err != nil {
			panic(err)
		}
	}

	var packages map[string]interface{}
	data, err := ioutil.ReadFile(packageManifest)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(data, &packages)

	searchRegex, err := cmd.Flags().GetString("search")
	if err != nil {
		panic(err)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"PackageName", "Version", "Language", "Description"})
	table.SetHeaderColor(
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

	for key, value := range packages {
		var row []string
		crow, _ := value.(map[string]interface{})
		// If we are told to filter and get no matches then filter out the
		// current row. If we are not told to filter then just add the
		// row.
		if filter &&
			!(r.MatchString(crow["language"].(string)) ||
				r.MatchString(crow["runtime"].(string)) ||
				r.MatchString(key)) {
			continue
		}
		row = append(row, key)
		row = append(row, crow["version"].(string))
		row = append(row, crow["language"].(string))
		row = append(row, crow["runtime"].(string))
		table.Append(row)
	}

	table.Render()
}

func main() {

	var cmdRun = &cobra.Command{
		Use:   "run [elf]",
		Short: "Run ELF binary as unikernel",
		Args:  cobra.MinimumNArgs(1),
		Run:   runCommandHandler,
	}

	var ports []string
	var force bool
	var debugflags bool
	var args []string
	var config string
	var verbose bool
	var bridged bool
	var nightly bool
	var search string
	var tap string
	var targetRoot string

	cmdRun.PersistentFlags().StringArrayVarP(&ports, "port", "p", nil, "port to forward")
	cmdRun.PersistentFlags().BoolVarP(&force, "force", "f", false, "update images")
	cmdRun.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")
	cmdRun.PersistentFlags().BoolVarP(&debugflags, "debug", "d", false, "enable all debug flags")
	cmdRun.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "command line arguments")
	cmdRun.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdRun.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	cmdRun.PersistentFlags().BoolVarP(&bridged, "bridged", "b", false, "bridge networking")
	cmdRun.PersistentFlags().StringVarP(&tap, "tapname", "t", "tap0", "tap device name")
	cmdRun.PersistentFlags().StringVarP(&targetRoot, "target-root", "r", "", "target root")

	var cmdNetSetup = &cobra.Command{
		Use:   "setup",
		Short: "Setup bridged network",
		Run:   netSetupCommandHandler,
	}

	var cmdNetReset = &cobra.Command{
		Use:   "reset",
		Short: "Reset bridged network",
		Run:   netResetCommandHandler,
	}

	var cmdNet = &cobra.Command{
		Use:       "net",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"setup", "reset"},
		Short:     "Configure bridge network",
	}

	var cmdBuild = &cobra.Command{
		Use:   "build [ELF file]",
		Short: "Build an image from ELF",
		Args:  cobra.MinimumNArgs(1),
		Run:   buildCommandHandler,
	}
	cmdBuild.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdBuild.PersistentFlags().StringVarP(&targetRoot, "target-root", "r", "", "target root")

	var cmdPrintConfig = &cobra.Command{
		Use:   "manifest [ELF file]",
		Short: "Print the manifest to console",
		Args:  cobra.MinimumNArgs(1),
		Run:   printManifestHandler,
	}
	cmdPrintConfig.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdPrintConfig.PersistentFlags().StringVarP(&targetRoot, "target-root", "r", "", "target root")

	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Version",
		Run:   printVersion,
	}

	var cmdList = &cobra.Command{
		Use:   "list",
		Short: "list packages",
		Run:   cmdListPackages,
	}
	cmdList.PersistentFlags().StringVarP(&search, "search", "s", "", "search package list")

	var cmdLoadPackage = &cobra.Command{
		Use:   "load [packagename]",
		Short: "load and run a package from ['ops list']",
		Args:  cobra.MinimumNArgs(1),
		Run:   loadCommandHandler,
	}
	cmdLoadPackage.PersistentFlags().StringArrayVarP(&ports, "port", "p", nil, "port to forward")
	cmdLoadPackage.PersistentFlags().BoolVarP(&force, "force", "f", false, "update images")
	cmdLoadPackage.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")
	cmdLoadPackage.PersistentFlags().BoolVarP(&debugflags, "debug", "d", false, "enable all debug flags")
	cmdLoadPackage.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "command line arguments")
	cmdLoadPackage.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdLoadPackage.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	cmdLoadPackage.PersistentFlags().BoolVarP(&bridged, "bridged", "b", false, "bridge networking")

	var cmdUpdate = &cobra.Command{
		Use:   "update",
		Short: "check for updates",
		Run:   updateCommandHandler,
	}

	var rootCmd = &cobra.Command{Use: "ops"}
	rootCmd.AddCommand(cmdRun)
	cmdNet.AddCommand(cmdNetReset)
	cmdNet.AddCommand(cmdNetSetup)
	rootCmd.AddCommand(cmdNet)
	rootCmd.AddCommand(cmdBuild)
	rootCmd.AddCommand(cmdPrintConfig)
	rootCmd.AddCommand(cmdVersion)
	rootCmd.AddCommand(cmdUpdate)
	rootCmd.AddCommand(cmdList)
	rootCmd.AddCommand(cmdLoadPackage)
	rootCmd.Execute()
}

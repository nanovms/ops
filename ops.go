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
			fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
			os.Exit(1)
		}
		err = json.Unmarshal(data, &c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error config: %v\n", err)
			os.Exit(1)
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

	skipbuild, err := strconv.ParseBool(cmd.Flag("skipbuild").Value.String())
	if err != nil {
		panic(err)
	}

	targetRoot, err := cmd.Flags().GetString("target-root")
	if err != nil {
		panic(err)
	}

	accel, err := cmd.Flags().GetBool("accel")
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
	c.RunConfig.UseKvm = accel
	c.NightlyBuild = nightly
	c.Force = force
	setDefaultImageName(cmd, c)

	if !skipbuild {
		buildImages(c)
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

	fmt.Printf("booting %s ...\n", c.RunConfig.Imagename)

	InitDefaultRunConfigs(c, ports)
	hypervisor.Start(&c.RunConfig)
}

func parseVersion(s string, width int) int64 {
	strList := strings.Split(s, ".")
	format := fmt.Sprintf("%%s%%0%ds", width)
	v := ""
	for _, value := range strList {
		v = fmt.Sprintf(format, v, value)
	}
	var result int64
	var err error
	if result, err = strconv.ParseInt(v, 10, 64); err != nil {
		panic(err)
	}
	return result
}

func downloadReleaseImages(c *api.Config) (string, error) {
	var err error
	// if it's first run or we have an update
	local, remote := api.LocalReleaseVersion, api.LatestReleaseVersion
	u := os.Getenv("NANOS_UPDATE")
	if local == "0.0" || (u == "1" && parseVersion(local, 4) != parseVersion(remote, 4)) {
		err = api.DownloadReleaseImages(remote)
		if err != nil {
			return "", err
		}
	}
	return remote, nil
}

func downloadNightlyImages(c *api.Config) (string, error) {
	var err error
	err = api.DownloadNightlyImages(c)
	return "nightly", err
}

func fixupConfigImages(c *api.Config, version string) {

	if c.NightlyBuild {
		version = "nightly"
	}

	if c.Boot == "" {
		c.Boot = path.Join(api.GetOpsHome(), version, "boot.img")
	}

	if c.Kernel == "" {
		c.Kernel = path.Join(api.GetOpsHome(), version, "stage3.img")
	}

	if c.Mkfs == "" {
		c.Mkfs = path.Join(api.GetOpsHome(), version, "mkfs")
	}

	if c.NameServer == "" {
		// google dns server
		c.NameServer = "8.8.8.8"
	}
}

func validateRequired(c *api.Config) {
	if _, err := os.Stat(c.Kernel); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(c.Mkfs); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(c.Boot); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(c.Program); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func prepareImages(c *api.Config) {
	var err error
	var currversion string

	if c.NightlyBuild {
		currversion, err = downloadNightlyImages(c)
	} else {
		currversion, err = downloadReleaseImages(c)
	}
	panicOnError(err)
	fixupConfigImages(c, currversion)
	validateRequired(c)
}

func buildImages(c *api.Config) {
	prepareImages(c)
	err := api.BuildImage(*c)
	panicOnError(err)
}

func setDefaultImageName(cmd *cobra.Command, c *api.Config) {
	// if user have not supplied an imagename, use the default as program_image
	// all images goes to $HOME/.ops/images
	imageName, _ := cmd.Flags().GetString("imagename")
	if imageName == "" {
		imageName = api.GenerateImageName(c.Program)
	} else {
		images := path.Join(api.GetOpsHome(), "images")
		imageName = path.Join(images, filepath.Base(imageName))
	}
	c.RunConfig.Imagename = imageName
	c.CloudConfig.ArchiveName = fmt.Sprintf("nanos-%v-image.tar.gz", filepath.Base(c.Program))
}

// TODO : use factory or DI
func getCloudProvider(c *api.Config, providerName string) api.Provider {
	var provider api.Provider
	if providerName == "gcp" {
		provider = &api.GCloud{}
	} else if providerName == "onprem" {
		provider = &api.OnPrem{}
	} else {
		fmt.Fprintf(os.Stderr, "error:Unknown provider %s", providerName)
		os.Exit(1)
	}
	provider.Initialize()
	return provider
}

func buildCommandHandler(cmd *cobra.Command, args []string) {
	targetRoot, _ := cmd.Flags().GetString("target-root")
	provider, _ := cmd.Flags().GetString("target-cloud")

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	c.Program = args[0]
	c.TargetRoot = targetRoot
	setDefaultImageName(cmd, c)

	p := getCloudProvider(c, provider)

	ctx := api.NewContext(c, &p)
	prepareImages(c)
	if _, err := p.BuildImage(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Bootable image file:%s\n", c.RunConfig.Imagename)
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

func imageCommandHandler(cmd *cobra.Command, args []string) {
	if _, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); !ok {
		fmt.Printf(api.ErrorColor, "error: GOOGLE_APPLICATION_CREDENTIALS not set.\n")
		fmt.Printf(api.ErrorColor, "Follow https://cloud.google.com/storage/docs/reference/libraries to set it up.\n")
		os.Exit(1)
	}

	provider, _ := cmd.Flags().GetString("target-cloud")
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)

	c := unWarpConfig(config)
	c.Program = args[0]

	// override config from command line
	if len(provider) > 0 {
		c.CloudConfig.Platform = provider
	}

	if len(c.CloudConfig.Platform) == 0 {
		fmt.Printf(api.ErrorColor, "error: Please select on of the cloud platform in config. [onprem, gcp]")
		os.Exit(1)
	}

	if len(c.CloudConfig.ProjectID) == 0 {
		fmt.Printf(api.ErrorColor, "error: Please specifiy a cloud projectid in config.\n")
		os.Exit(1)
	}

	if len(c.CloudConfig.BucketName) == 0 {
		fmt.Printf(api.ErrorColor, "error: Please specifiy a cloud bucket in config.\n")
		os.Exit(1)
	}

	setDefaultImageName(cmd, c)

	p := getCloudProvider(c, provider)
	ctx := api.NewContext(c, &p)
	prepareImages(c)

	archpath, err := p.BuildImage(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	gcloud := p.(*api.GCloud)
	err = gcloud.Storage.CopyToBucket(c, archpath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = gcloud.CreateImage(ctx)
	if err != nil {
		fmt.Println(err)
	} else {
		imageName := fmt.Sprintf("nanos-%v-image", filepath.Base(c.Program))
		fmt.Printf("gcp image '%s' created...\n", imageName)
	}
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

	pkgConfig.RunConfig = usrConfig.RunConfig
	pkgConfig.Kernel = usrConfig.Kernel
	pkgConfig.Boot = usrConfig.Boot
	pkgConfig.Mkfs = usrConfig.Mkfs
	pkgConfig.TargetRoot = usrConfig.TargetRoot
	pkgConfig.Force = usrConfig.Force
	pkgConfig.NightlyBuild = usrConfig.NightlyBuild
	pkgConfig.NameServer = usrConfig.NameServer

	return pkgConfig
}

func buildFromPackage(packagepath string, c *api.Config) error {
	var err error
	var currversion string

	if c.NightlyBuild {
		currversion, err = downloadNightlyImages(c)
	} else {
		currversion, err = downloadReleaseImages(c)
	}
	panicOnError(err)
	fixupConfigImages(c, currversion)
	return api.BuildImageFromPackage(packagepath, *c)
}

func loadCommandHandler(cmd *cobra.Command, args []string) {

	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}

	localstaging := path.Join(api.GetOpsHome(), ".staging")
	err := os.MkdirAll(localstaging, 755)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	expackage := path.Join(localstaging, args[0])
	localpackage := api.DownloadPackage(args[0])

	fmt.Printf("Extracting %s to %s\n", localpackage, expackage)

	// Remove the folder first.
	os.RemoveAll(expackage)
	api.ExtractPackage(localpackage, localstaging)

	// load the package manifest
	manifest := path.Join(expackage, "package.manifest")
	if _, err := os.Stat(manifest); err != nil {
		panic(err)
	}

	pkgConfig := unWarpConfig(manifest)

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

	accel, err := cmd.Flags().GetBool("accel")
	if err != nil {
		panic(err)
	}

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	cmdargs, _ := cmd.Flags().GetStringArray("args")
	c := unWarpConfig(config)
	c.Args = append(c.Args, cmdargs...)

	if debugflags {
		pkgConfig.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}

	c = mergeConfigs(pkgConfig, c)
	pkgConfig.RunConfig.Verbose = verbose
	pkgConfig.RunConfig.Bridged = bridged
	pkgConfig.NightlyBuild = nightly
	pkgConfig.Force = force
	pkgConfig.RunConfig.UseKvm = accel
	setDefaultImageName(cmd, c)

	if err = buildFromPackage(expackage, c); err != nil {
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

	fmt.Printf("booting %s ...\n", c.RunConfig.Imagename)
	InitDefaultRunConfigs(c, ports)
	hypervisor.Start(&c.RunConfig)
}

func InitDefaultRunConfigs(c *api.Config, ports []int) {

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

// PackageList contains a list of known packages.
type PackageList struct {
	list map[string]Package
}

// Package is the definition of an OPS package.
type Package struct {
	Runtime     string `json:"runtime"`
	Version     string `json:"version"`
	Language    string `json:"language"`
	Description string `json:"description,omitempty"`
	MD5         string `json:"md5,omitempty"`
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

	var packages PackageList
	data, err := ioutil.ReadFile(packageManifest)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &packages.list)
	if err != nil {
		fmt.Println(err)
	}

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

	for key, val := range packages.list {
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
	var targetCloud string
	var skipbuild bool
	var imageName string
	var accel bool

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
	cmdRun.PersistentFlags().BoolVarP(&skipbuild, "skipbuild", "s", false, "skip building image")
	cmdRun.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	cmdRun.PersistentFlags().BoolVarP(&accel, "accel", "x", false, "use cpu virtualization extension")

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
	cmdBuild.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform[gcp, onprem]")
	cmdBuild.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")

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
	cmdLoadPackage.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	cmdLoadPackage.PersistentFlags().BoolVarP(&accel, "accel", "x", false, "use cpu virtualization extension")

	var cmdUpdate = &cobra.Command{
		Use:   "update",
		Short: "check for updates",
		Run:   updateCommandHandler,
	}

	var cmdImageCreate = &cobra.Command{
		Use:   "create",
		Short: "create nanos image from ELF",
		Args:  cobra.MinimumNArgs(1),
		Run:   imageCommandHandler,
	}

	cmdImageCreate.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdImageCreate.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")

	var cmdImage = &cobra.Command{
		Use:       "image",
		Short:     "manage nanos images",
		ValidArgs: []string{"create"},
		Args:      cobra.OnlyValidArgs,
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

	cmdImage.AddCommand(cmdImageCreate)

	rootCmd.AddCommand(cmdImage)
	rootCmd.Execute()
}

package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	api "github.com/nanovms/ops/lepton"
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

	force, err := strconv.ParseBool(cmd.Flag("force").Value.String())
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

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	cmdargs, _ := cmd.Flags().GetStringArray("args")
	c := unWarpConfig(config)
	c.Args = append(c.Args, cmdargs...)
	c.Program = args[0]
	if debugflags {
		c.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}
	c.RunConfig.Verbose = verbose
	c.RunConfig.Bridged = bridged
	c.NightlyBuild = force

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

	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}
	InitDefaultRunConfigs(c, ports)
	hypervisor.Start(&c.RunConfig)
}

func buildImages(c *api.Config) {
	var err error
	if c.NightlyBuild {
		err = api.DownloadImages(api.DevBaseUrl, c.NightlyBuild)
	} else {
		err = api.DownloadImages(api.ReleaseBaseUrl, c.NightlyBuild)
	}
	panicOnError(err)
	err = api.BuildImage(*c)
	panicOnError(err)
}

func buildCommandHandler(cmd *cobra.Command, args []string) {
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	buildImages(c)
}

func printManifestHandler(cmd *cobra.Command, args []string) {
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	c.Program = args[0]
	m, err := api.BuildManifest(c)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(m.String())
}

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Println("0.1")
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

func buildFromPackage(packagepath string, c *api.Config) {
	var err error
	if c.NightlyBuild {
		err = api.DownloadImages(api.DevBaseUrl, c.NightlyBuild)
	} else {
		err = api.DownloadImages(api.ReleaseBaseUrl, c.NightlyBuild)
	}
	panicOnError(err)
	err = api.BuildImageFromPackage(packagepath, *c)
	panicOnError(err)
}

// merge userconfig to package config, user config takes precedence
func mergeConfigs(pkgConfig *api.Config, usrConfig *api.Config) *api.Config {

	pkgConfig.Args = append(pkgConfig.Args, usrConfig.Args...)
	pkgConfig.Dirs = append(pkgConfig.Dirs, usrConfig.Dirs...)
	pkgConfig.Files = append(pkgConfig.Files, usrConfig.Files...)

	pkgConfig.Debugflags = usrConfig.Debugflags
	pkgConfig.RunConfig.Verbose = usrConfig.RunConfig.Verbose

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

	localpackage := DownloadPackage(args[0])
	fmt.Printf("Extracting %s...\n", localpackage)
	ExtractPackage(localpackage, ".staging")

	// load the package manifest
	manifest := path.Join(".staging", args[0], "package.manifest")
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

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	cmdargs, _ := cmd.Flags().GetStringArray("args")
	c := unWarpConfig(config)
	c.Args = append(c.Args, cmdargs...)
	c.RunConfig.Verbose = verbose
	c.RunConfig.Bridged = bridged
	c.NightlyBuild = force
	if debugflags {
		c.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}

	c = mergeConfigs(pkgConfig, c)

	buildFromPackage(path.Join(".staging", args[0]), c)

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
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}

	InitDefaultRunConfigs(c, ports)
	hypervisor.Start(&c.RunConfig)

}

func InitDefaultRunConfigs(c *api.Config, ports []int) {

	if c.RunConfig.Imagename == "" {
		c.RunConfig.Imagename = api.FinalImg
	}
	if c.RunConfig.Memory == "" {
		c.RunConfig.Memory = "2G"
	}
	c.RunConfig.Ports = append(c.RunConfig.Ports, ports...)
}

func DownloadPackage(name string) string {

	archivename := name + ".tar.gz"
	packagepath := path.Join(api.GetPackageCache(), archivename)
	if _, err := os.Stat(packagepath); os.IsNotExist(err) {
		if err = api.DownloadFile(packagepath,
			fmt.Sprintf(api.PackageBaseURL, archivename)); err != nil {
			panic(err)
		}
	}
	return packagepath
}

func ExtractPackage(archive string, dest string) {
	in, err := os.Open(archive)
	if err != nil {
		panic(err)
	}
	gzip, err := gzip.NewReader(in)
	if err != nil {
		panic(err)
	}
	defer gzip.Close()
	tr := tar.NewReader(gzip)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return
		}
		if err != nil {
			panic(err)
		}
		if header == nil {
			continue
		}
		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					panic(err)
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				panic(err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				panic(err)
			}
			f.Close()
		}
	}
}

func cmdListPackages(cmd *cobra.Command, args []string) {

	var err error
	if _, err := os.Stat(".staging"); os.IsNotExist(err) {
		os.MkdirAll(".staging", 0755)
	}

	if _, err = os.Stat(api.PackageManifest); os.IsNotExist(err) {
		if err = api.DownloadFile(api.PackageManifest, api.PackageManifestURL); err != nil {
			panic(err)
		}
	}

	var pacakges map[string]interface{}
	data, err := ioutil.ReadFile(api.PackageManifest)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(data, &pacakges)
	for key := range pacakges {
		fmt.Println(key)
		/*
			value, _ := value.(map[string]interface{})
			fmt.Print(value["runtime"])
			fmt.Print(value["version"])
		*/
	}

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

	cmdRun.PersistentFlags().StringArrayVarP(&ports, "port", "p", nil, "port to forward")
	cmdRun.PersistentFlags().BoolVarP(&force, "force", "f", false, "nightly build")
	cmdRun.PersistentFlags().BoolVarP(&debugflags, "debug", "d", false, "enable all debug flags")
	cmdRun.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "commanline arguments")
	cmdRun.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdRun.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	cmdRun.PersistentFlags().BoolVarP(&bridged, "bridged", "b", false, "bridge networking")

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
	var cmdPrintConfig = &cobra.Command{
		Use:   "manifest [ELF file]",
		Short: "Print the manifest to console",
		Args:  cobra.MinimumNArgs(1),
		Run:   printManifestHandler,
	}
	cmdPrintConfig.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")

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

	var cmdLoadPackage = &cobra.Command{
		Use:   "load [packagename]",
		Short: "load and run a package from ['ops list']",
		Args:  cobra.MinimumNArgs(1),
		Run:   loadCommandHandler,
	}

	cmdLoadPackage.PersistentFlags().StringArrayVarP(&ports, "port", "p", nil, "port to forward")
	cmdLoadPackage.PersistentFlags().BoolVarP(&debugflags, "debug", "d", false, "enable all debug flags")
	cmdLoadPackage.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "commanline arguments")
	cmdLoadPackage.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdLoadPackage.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	cmdLoadPackage.PersistentFlags().BoolVarP(&bridged, "bridged", "b", false, "bridge networking")
	cmdLoadPackage.PersistentFlags().BoolVarP(&force, "force", "f", false, "nightly build")

	var rootCmd = &cobra.Command{Use: "ops"}
	rootCmd.AddCommand(cmdRun)
	cmdNet.AddCommand(cmdNetReset)
	cmdNet.AddCommand(cmdNetSetup)
	rootCmd.AddCommand(cmdNet)
	rootCmd.AddCommand(cmdBuild)
	rootCmd.AddCommand(cmdPrintConfig)
	rootCmd.AddCommand(cmdVersion)
	rootCmd.AddCommand(cmdList)
	rootCmd.AddCommand(cmdLoadPackage)
	rootCmd.Execute()
}

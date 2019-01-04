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

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	cmdargs, _ := cmd.Flags().GetStringArray("args")
	c := unWarpConfig(config)
	c.Args = append(c.Args, cmdargs...)
	if debugflags {
		c.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}
	buildImages(args[0], force, c)
	fmt.Printf("booting %s ...\n", api.FinalImg)
	port, err := strconv.Atoi(cmd.Flag("port").Value.String())
	if err != nil {
		panic(err)
	}
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}
	runconfig := api.RuntimeConfig(api.FinalImg, []int{port}, verbose)
	hypervisor.Start(&runconfig)
}

func buildImages(userBin string, useLatest bool, c *api.Config) {
	var err error
	if useLatest {
		err = api.DownloadImages(callback{}, api.DevBaseUrl)
	} else {
		err = api.DownloadImages(callback{}, api.ReleaseBaseUrl)
	}
	panicOnError(err)
	err = api.BuildImage(userBin, *c)
	panicOnError(err)
}

func buildCommandHandler(cmd *cobra.Command, args []string) {
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	buildImages(args[0], false, c)
}

func printManifestHandler(cmd *cobra.Command, args []string) {
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	m, err := api.BuildManifest(args[0], c)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(m.String())
}

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Println("0.1")
}

type callback struct {
	total uint64
}

func (bc callback) Write(p []byte) (int, error) {
	n := len(p)
	bc.total += uint64(n)
	bc.printProgress()
	return n, nil
}

func (bc callback) printProgress() {
	// clear the previous line
	fmt.Printf("\r%s", strings.Repeat(" ", 35))
	fmt.Printf("\rDownloading... %v complete", bc.total)
}

func runningAsRoot() bool {
	cmd := exec.Command("id", "-u")
	output, _ := cmd.Output()
	i, _ := strconv.Atoi(string(output[:len(output)-1]))
	return i == 0
}

func netCommandHandler(cmd *cobra.Command, args []string) {
	if !runningAsRoot() {
		fmt.Println("net command needs root permission")
		return
	}
	if len(args) < 1 {
		fmt.Println("Not enough arguments.")
		return
	}
	if args[0] == "setup" {
		if err := setupBridgeNetwork(); err != nil {
			panic(err)
		}
	} else {
		if err := resetBridgeNetwork(); err != nil {
			panic(err)
		}
	}
}

func buildFromPackage(program string, packagepath string, c *api.Config) {
	var err error
	err = api.DownloadImages(callback{}, api.ReleaseBaseUrl)
	panicOnError(err)
	err = api.BuildImageFromPackage(program, packagepath, *c)
	panicOnError(err)
}

func loadCommandHandler(cmd *cobra.Command, args []string) {

	localpackage := DownloadPackage(args[0])
	ExtractPackage(localpackage, ".staging")
	// load the package manifest
	manifest := path.Join(".staging", args[0], "package.manifest")
	if _, err := os.Stat(manifest); err != nil {
		panic(err)
	}
	data, err := ioutil.ReadFile(manifest)
	if err != nil {
		panic(err)
	}
	var metadata map[string]string
	json.Unmarshal(data, &metadata)
	program := path.Join(args[0], metadata["program"])

	debugflags, err := strconv.ParseBool(cmd.Flag("debug").Value.String())
	if err != nil {
		panic(err)
	}

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	cmdargs, _ := cmd.Flags().GetStringArray("args")
	c := unWarpConfig(config)
	c.Args = append(c.Args, cmdargs...)
	if debugflags {
		c.Debugflags = []string{"trace", "debugsyscalls", "futex_trace", "fault"}
	}

	buildFromPackage(program, path.Join(".staging", args[0]), c)

	fmt.Printf("booting %s ...\n", api.FinalImg)
	port, err := strconv.Atoi(cmd.Flag("port").Value.String())
	if err != nil {
		panic(err)
	}
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		panic(errors.New("No hypervisor found on $PATH"))
	}
	runconfig := api.RuntimeConfig(api.FinalImg, []int{port}, true)
	hypervisor.Start(&runconfig)
}

func DownloadPackage(name string) string {

	if _, err := os.Stat(api.PackagesCache); os.IsNotExist(err) {
		os.MkdirAll(api.PackagesCache, 0755)
	}
	archivename := name + ".tar.gz"
	packagepath := path.Join(api.PackagesCache, archivename)
	if _, err := os.Stat(packagepath); os.IsNotExist(err) {
		if err = api.DownloadFile(packagepath, fmt.Sprintf(api.PackageBaseURL, archivename)); err != nil {
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

	var port int
	var force bool
	var debugflags bool
	var args []string
	var config string
	var verbose bool

	cmdRun.PersistentFlags().IntVarP(&port, "port", "p", -1, "port to forward")
	cmdRun.PersistentFlags().BoolVarP(&force, "force", "f", false, "use latest dev images")
	cmdRun.PersistentFlags().BoolVarP(&debugflags, "debug", "d", false, "enable all debug flags")
	cmdRun.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "commanline arguments")
	cmdRun.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdRun.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")

	var cmdNet = &cobra.Command{
		Use:       "net",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"setup", "reset"},
		Short:     "Configure bridge network",
		Run:       netCommandHandler,
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

	cmdLoadPackage.PersistentFlags().IntVarP(&port, "port", "p", -1, "port to forward")
	cmdLoadPackage.PersistentFlags().BoolVarP(&debugflags, "debug", "d", false, "enable all debug flags")
	cmdLoadPackage.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "commanline arguments")
	cmdLoadPackage.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")

	var rootCmd = &cobra.Command{Use: "ops"}
	rootCmd.AddCommand(cmdRun)
	rootCmd.AddCommand(cmdNet)
	rootCmd.AddCommand(cmdBuild)
	rootCmd.AddCommand(cmdPrintConfig)
	rootCmd.AddCommand(cmdVersion)
	rootCmd.AddCommand(cmdList)
	rootCmd.AddCommand(cmdLoadPackage)
	rootCmd.Execute()
}

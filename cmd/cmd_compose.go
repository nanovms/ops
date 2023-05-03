package cmd

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"time"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider/onprem"
	"github.com/nanovms/ops/qemu"
	"github.com/nanovms/ops/types"

	"github.com/spf13/cobra"
	"net/http"

	"gopkg.in/yaml.v2"
)

// the following starts a compose session
//
// (eg: starts a dns server for svc discovery && associated unikernels
// in same network; each unikernel gets an address of name.service)
//
// ops compose up
//
// experimental and objective should be to *not* interfere with existing
// 'ops run', 'ops pkg load', 'ops instance create' functionality
//
// run first:
//	ops pkg get eyberg/ops-dns:0.0.1
//
// compose.yaml:
//
// packages:
//  - pkg: myserver
//    name: mynewserver:0.0.1
//  - pkg: myclient
//    name: mynewclient:0.0.1
//
//
// much of this probably belongs in a diff. pkg but not sure what to do
// there yet

// ComposeCommands provides support for running binary with nanos
func ComposeCommands() *cobra.Command {

	var cmdCompose = &cobra.Command{
		Use:       "compose",
		Short:     "orchestrate multiple unikernels",
		ValidArgs: []string{"up", "down"},
		Args:      cobra.OnlyValidArgs,
	}

	cmdCompose.AddCommand(composeUpCommand())
	cmdCompose.AddCommand(composeDownCommand())

	return cmdCompose
}

func composeDownCommand() *cobra.Command {
	var cmdDownCompose = &cobra.Command{
		Use:   "down",
		Short: "spin unikernels down",
		Run:   composeDownCommandHandler,
	}

	return cmdDownCompose
}

func composeUpCommand() *cobra.Command {
	var cmdUpCompose = &cobra.Command{
		Use:   "up",
		Short: "spin unikernels up",
		Run:   composeUpCommandHandler,
	}

	return cmdUpCompose
}

// Package is a part of the compose yaml file.
type Package struct {
	Pkg  string
	Name string
}

// ComposeFile represents a configuration for ops compose.
type ComposeFile struct {
	Packages []Package
}

func genNon(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length+2)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}

func composeDownCommandHandler(cmd *cobra.Command, args []string) {
	if qemu.OPSD == "" {
		fmt.Println("this command is only enabled if you have OPSD compiled in.")
		os.Exit(1)
	}

	c := api.NewConfig()

	p, ctx, err := getProviderAndContext(c, "onprem")
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	instances, err := p.GetInstances(ctx)
	if err != nil {
		fmt.Println(err)
	}

	for i := 0; i < len(instances); i++ {
		err = p.DeleteInstance(ctx, instances[i].Name)
		if err != nil {
			exitWithError(err.Error())
		}
	}
}

func composeUpCommandHandler(cmd *cobra.Command, args []string) {
	if qemu.OPSD == "" {
		fmt.Println("this command is only enabled if you have OPSD compiled in.")
		os.Exit(1)
	}

	fmt.Println("running compose")

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}

	body, err := os.ReadFile(dir + "/compose.yaml")
	if err != nil {
		fmt.Println(err)
	}

	y := ComposeFile{}

	err = yaml.Unmarshal(body, &y)
	if err != nil {
		fmt.Println(err)
	}

	non := genNon(32)
	pid := spawnDNS(non)

	dnsIP, err := waitForIP(pid)
	if err != nil {
		fmt.Println(err)
	}

	// FIXME
	c := api.NewConfig()
	version := api.LocalReleaseVersion
	c.Boot = path.Join(api.GetOpsHome(), version, "boot.img")
	c.Kernel = path.Join(api.GetOpsHome(), version, "kernel.img")

	// spawn other pkgs
	for i := 0; i < len(y.Packages); i++ {
		pid := spawnProgram(y.Packages[i].Name, y.Packages[i].Pkg, dnsIP, c)
		ip, err := waitForIP(pid)
		if err != nil {
			fmt.Println(err)
		}

		addDNS(dnsIP, y.Packages[i].Pkg, ip, non)
	}
}

func waitForIP(pid string) (string, error) {
	ip := ""
	for i := 0; i < 10; i++ {
		ip = onprem.FindBridgedIPByPID(pid)
		if ip == "" {
			if 1 == 2 {
				fmt.Println("no ip found")
			}
			time.Sleep(time.Millisecond * 500)
		} else {
			if 1 == 2 {
				fmt.Printf("found ip of %s\n", ip)
			}
			return ip, nil
		}
	}

	return "", errors.New("ip timeout")
}

func addDNS(dnsIP string, host string, ip string, non string) {
	client := &http.Client{}

	fmt.Printf("adding record %s for %s\n", host, ip)
	req, err := http.NewRequest("GET", "http://"+dnsIP+":8080/add?svc="+host+".service&ip="+ip, nil)
	if err != nil {
		fmt.Println(err)
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, error := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(error)
	}

	fmt.Println(string(body))
}

func spawnProgram(pkgName string, pname string, dnsIP string, c *types.Config) string {
	fmt.Printf("spawing %s\n", pkgName)

	pkgFlags := PkgCommandFlags{
		Package: pkgName,
	}

	ppath := filepath.Join(pkgFlags.PackagePath()) + "/package.manifest"
	unWarpConfig(ppath, c)

	executableName := c.Program

	c.RunConfig.ImageName = path.Join(api.GetOpsHome(), "images", pname)
	c.RunConfig.InstanceName = pname

	c.NameServers = []string{dnsIP}

	api.ValidateELF(filepath.Join(api.GetOpsHome(), "packages", executableName))

	p, ctx, err := getProviderAndContext(c, "onprem")
	if err != nil {
		fmt.Println(err)
	}

	var keypath string
	if pkgFlags.Package != "" {
		keypath, err = p.BuildImageWithPackage(ctx, pkgFlags.PackagePath())
		if err != nil {
			exitWithError(err.Error())
		}
	} else {
		keypath, err = p.BuildImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		}
	}

	err = p.CreateImage(ctx, keypath)
	if err != nil {
		exitWithError(err.Error())
	}

	c.RunConfig.InstanceName = pname
	c.CloudConfig.ImageName = pname

	z := p.(*onprem.OnPrem)
	pid, err := z.CreateInstancePID(ctx)
	if err != nil {
		exitWithError(err.Error())
	}

	fmt.Printf("%s instance with pid %s '%s' created...\n", c.CloudConfig.Platform, pid, c.RunConfig.InstanceName)

	return pid
}

func spawnDNS(non string) string {
	c := api.NewConfig()
	c.Program = "dns"

	pkgFlags := PkgCommandFlags{
		Package: "eyberg/ops-dns:0.0.1",
	}

	executableName := c.Program

	api.ValidateELF(filepath.Join(pkgFlags.PackagePath(), executableName))

	p, ctx, err := getProviderAndContext(c, "onprem")
	if err != nil {
		fmt.Println(err)
	}

	c.RunConfig.InstanceName = "dns"
	c.CloudConfig.ImageName = "dns"

	env := map[string]string{"non": non}

	c.Env = env

	z := p.(*onprem.OnPrem)
	pid, err := z.CreateInstancePID(ctx)
	if err != nil {
		exitWithError(err.Error())
	}

	fmt.Printf("%s instance with pid %s '%s' created...\n", c.CloudConfig.Platform, pid, c.RunConfig.InstanceName)

	return pid
}

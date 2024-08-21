package cmd

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider/onprem"
	"github.com/nanovms/ops/types"

	"net/http"

	"gopkg.in/yaml.v2"
)

// Compose holds information related to a compose session.
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
//
//	ops pkg get eyberg/ops-dns:0.0.1
//
// compose.yaml:
//
// packages:
//   - pkg: myserver
//     name: mynewserver:0.0.1
//   - pkg: myclient
//     name: mynewclient:0.0.1
//
// much of this probably belongs in a diff. pkg but not sure what to do
// there yet
type Compose struct {
	config *types.Config // don't think this belongs here
}

// UP reads in a compose.yaml and starts all services listed with svc
// discovery.
func (com Compose) UP(composeFile string) {

	if composeFile == "" {

		dir, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
		}

		composeFile = dir + "/compose.yaml"
	}

	body, err := os.ReadFile(composeFile)
	if err != nil {
		fmt.Println(err)
		fmt.Println("are you running compose in the same directory as a compose.yaml?")
		os.Exit(1)
	}

	y := ComposeFile{}

	err = yaml.Unmarshal(body, &y)
	if err != nil {
		fmt.Println(err)
	}

	non := genNon(32)
	pid := com.spawnDNS(non)

	dnsIP, err := com.waitForIP(pid)
	if err != nil {
		fmt.Println(err)
	}

	// FIXME
	version := api.LocalReleaseVersion
	com.config.Boot = path.Join(api.GetOpsHome(), version, "boot.img")

	// spawn other pkgs
	for i := 0; i < len(y.Packages); i++ {
		pid := com.spawnProgram(y.Packages[i], dnsIP, com.config)
		ip, err := com.waitForIP(pid)
		if err != nil {
			fmt.Println(err)
		}

		com.addDNS(dnsIP, y.Packages[i].Pkg, ip, non)
	}
}

func (com Compose) waitForIP(pid string) (string, error) {
	ip := ""
	for i := 0; i < 10; i++ {
		ip = onprem.FindBridgedIPByPID(pid)
		if ip == "" {
			if com.config.RunConfig.ShowDebug {

				fmt.Println("no ip found")
			}
			time.Sleep(time.Millisecond * 500)
		} else {
			if com.config.RunConfig.ShowDebug {
				fmt.Printf("found ip of %s\n", ip)
			}
			return ip, nil
		}
	}

	return "", errors.New("ip timeout")
}

func (com Compose) addDNS(dnsIP string, host string, ip string, non string) {
	client := &http.Client{}
	if com.config.RunConfig.ShowDebug {
		fmt.Printf("adding record %s for %s\n", host, ip)
	}
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

	if com.config.RunConfig.ShowDebug {
		fmt.Println(string(body))
	}
}

func (com Compose) spawnProgram(comp ComposePackage, dnsIP string, c *types.Config) string {

	pkgName := comp.Name
	pname := comp.Pkg
	local := comp.Local
	arch := comp.Arch
	baseVolumeSz := comp.BaseVolumeSz

	api.AltGOARCH = arch // this isn't really mt-safe..

	pkgFlags := PkgCommandFlags{
		Package:      pkgName,
		LocalPackage: local,
	}

	ppath := filepath.Join(pkgFlags.PackagePath()) + "/package.manifest"
	if local {
		ppath = strings.ReplaceAll(ppath, ":", "_")
	}

	unWarpConfig(ppath, c)

	c.BaseVolumeSz = baseVolumeSz

	// we need to reset this for each instance in the compose.
	// FIXME: this is not mt-safe at all; eventually api.AltGOARCH needs
	// to be instantiated per instance - it's just that every other
	// operation in ops assumes a single event.
	version, err := getCurrentVersion()
	if err != nil {
		fmt.Println(err)
	}
	version = setKernelVersion(version)
	c.Kernel = getKernelVersion(version)
	c.RunConfig.Kernel = c.Kernel

	executableName := c.Program

	c.RunConfig.ImageName = path.Join(api.GetOpsHome(), "images", pname)
	c.RunConfig.InstanceName = pname

	c.NameServers = []string{dnsIP}

	packageFolder := filepath.Base(pkgFlags.PackagePath())
	if strings.Contains(executableName, packageFolder) {
		executableName = filepath.Base(executableName)
	} else {
		executableName = filepath.Join(api.PackageSysRootFolderName, executableName)
	}

	p, ctx, err := getProviderAndContext(c, "onprem")
	if err != nil {
		fmt.Println(err)
	}

	// really shouldn't be hitting this..
	if ctx.Config().RunConfig.Kernel == "" {
		ctx.Config().RunConfig.Kernel = c.Kernel
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

	if c.RunConfig.ShowDebug {
		fmt.Printf("%s instance with pid %s '%s' created...\n", c.CloudConfig.Platform, pid, c.RunConfig.InstanceName)
	}

	return pid
}

// spawnDNS will grab whatever native pkg exists for the platform.
// no need to set a custom one.
func (com Compose) spawnDNS(non string) string {
	c := api.NewConfig()
	c.Program = "dns"

	version := api.LocalReleaseVersion
	c.Boot = path.Join(api.GetOpsHome(), version, "boot.img")
	c.RunConfig.ImageName = path.Join(api.GetOpsHome(), "images", "dns")

	// ideally all of this should happen in one place
	if c.Kernel == "" {
		version, err := getCurrentVersion()
		if err != nil {
			fmt.Println(err)
		}
		version = setKernelVersion(version)

		c.Kernel = getKernelVersion(version)

		c.RunConfig.Kernel = c.Kernel
	}

	pkgFlags := PkgCommandFlags{
		Package: "eyberg/ops-dns:0.0.1",
	}

	ppath := filepath.Join(pkgFlags.PackagePath()) + "/package.manifest"

	_, err := os.Stat(ppath)
	if err != nil {
		fmt.Println(err)
		fmt.Println("you need the dns package to use compose\ndownload it via:\n\tops pkg get eyberg/ops-dns:0.0.1")
		os.Exit(1)
	}

	unWarpConfig(ppath, c)

	packageFolder := filepath.Base(pkgFlags.PackagePath())
	executableName := c.Program
	if strings.Contains(executableName, packageFolder) {
		executableName = filepath.Base(executableName)
	} else {
		executableName = filepath.Join(api.PackageSysRootFolderName, executableName)
	}

	api.ValidateELF(filepath.Join(pkgFlags.PackagePath(), executableName))

	p, ctx, err := getProviderAndContext(c, "onprem")
	if err != nil {
		fmt.Println(err)
	}

	c.RunConfig.InstanceName = "dns"

	keypath, err := p.BuildImageWithPackage(ctx, pkgFlags.PackagePath())
	if err != nil {
		fmt.Println(err)
	}

	err = p.CreateImage(ctx, keypath)
	if err != nil {
		exitWithError(err.Error())
	}

	c.CloudConfig.ImageName = "dns"

	env := map[string]string{"non": non}

	c.Env = env

	z := p.(*onprem.OnPrem)
	pid, err := z.CreateInstancePID(ctx)
	if err != nil {
		exitWithError(err.Error())
	}

	if c.RunConfig.ShowDebug {
		fmt.Printf("%s instance with pid %s '%s' created...\n", c.CloudConfig.Platform, pid, c.RunConfig.InstanceName)
	}

	return pid
}

// ComposePackage is a part of the compose yaml file.
type ComposePackage struct {
	Pkg          string
	Name         string
	Local        bool
	Arch         string
	BaseVolumeSz string `yaml:"base_volume_sz"`
}

// ComposeFile represents a configuration for ops compose.
type ComposeFile struct {
	Packages []ComposePackage
}

func genNon(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length+2)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}

package cmd

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider/onprem"
	"github.com/nanovms/ops/types"

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

func getComposeContents(composeFile string) []byte {
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

	return body
}

func genBridgeName() string {
	return "ops0"
}

func (com Compose) validatePackagesExist(y ComposeFile) {

	for i := 0; i < len(y.Packages); i++ {
		comp := y.Packages[i]

		pkgName := comp.Name
		local := comp.Local
		arch := comp.Arch

		api.AltGOARCH = arch // this isn't really mt-safe..

		pkgFlags := PkgCommandFlags{
			Package:      pkgName,
			LocalPackage: local,
		}

		if !local {
			ppath := filepath.Join(pkgFlags.PackagePath()) + "/package.manifest"

			_, err := os.Stat(ppath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("%s not found - downloading\n", pkgName)
					downloadPackage(pkgName, &types.Config{})
				}
			}
		}

	}
}

// UP reads in a compose.yaml and starts all services listed with svc
// discovery.
func (com Compose) UP(composeFile string) error {

	body := getComposeContents(composeFile)

	h := sha1.New()
	h.Write([]byte(body))
	sha := hex.EncodeToString(h.Sum(nil))

	y, err := parseCompose(body)
	if err != nil {
		return err
	}

	com.validatePackagesExist(y)

	brName := genBridgeName()

	opshome := api.GetOpsHome()
	composes := path.Join(opshome, "composes")

	err = os.WriteFile(composes+"/"+sha, []byte(brName), 0644)
	if err != nil {
		fmt.Println(err)
	}

	non := genNon(32)
	pid, err := com.spawnDNS(non, brName)
	if err != nil {
		return err
	}

	dnsIP, err := com.waitForIP(pid)
	if err != nil {
		return err
	}

	// FIXME
	version := api.LocalReleaseVersion
	com.config.Boot = path.Join(api.GetOpsHome(), version, "boot.img")

	// spawn other pkgs
	for i := 0; i < len(y.Packages); i++ {
		pid, err := com.spawnProgram(y.Packages[i], dnsIP, y.DNSSearch, com.config, brName)
		if err != nil {
			return err
		}

		ip, err := com.waitForIP(pid)
		if err != nil {
			return err
		}

		err = com.addDNS(dnsIP, y.Packages[i].Pkg, ip, non)
		if err != nil {
			return err
		}
	}

	return nil
}

func (com Compose) waitForIP(pid string) (string, error) {
	ip := ""
	for i := 0; i < 20; i++ {
		ip = onprem.FindBridgedIPByPID(pid)
		if ip == "" {
			time.Sleep(time.Millisecond * 500)
		} else {
			return ip, nil
		}
	}

	return "", errors.New("ip timeout")
}

func (com Compose) addDNS(dnsIP string, host string, ip string, non string) error {
	client := &http.Client{}
	if com.config.RunConfig.ShowDebug {
		fmt.Printf("adding record %s for %s\n", host, ip)
	}
	req, err := http.NewRequest("GET", "http://"+dnsIP+":8080/add?svc="+host+".service&ip="+ip, nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	_, err = io.ReadAll(res.Body)
	return err
}

func (com Compose) spawnProgram(comp ComposePackage, dnsIP string, search []string, c *types.Config, brName string) (string, error) {

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

	if baseVolumeSz != "" {
		c.BaseVolumeSz = baseVolumeSz
	}

	// we need to reset this for each instance in the compose.
	// FIXME: this is not mt-safe at all; eventually api.AltGOARCH needs
	// to be instantiated per instance - it's just that every other
	// operation in ops assumes a single event.
	version, err := getCurrentVersion()
	if err != nil {
		return "", err
	}
	version = setKernelVersion(version)
	c.Kernel = getKernelVersion(version)
	c.RunConfig.Kernel = c.Kernel

	executableName := c.Program

	c.RunConfig.ImageName = path.Join(api.GetOpsHome(), "images", pname)
	c.RunConfig.InstanceName = pname

	setResolver(c, dnsIP, search)

	packageFolder := filepath.Base(pkgFlags.PackagePath())
	if strings.Contains(executableName, packageFolder) {
		executableName = filepath.Base(executableName)
	} else {
		executableName = filepath.Join(api.PackageSysRootFolderName, executableName)
	}

	p, ctx, err := getProviderAndContext(c, "onprem")
	if err != nil {
		return "", err
	}

	// really shouldn't be hitting this..
	if ctx.Config().RunConfig.Kernel == "" {
		ctx.Config().RunConfig.Kernel = c.Kernel
	}

	var keypath string
	if pkgFlags.Package != "" {
		keypath, err = p.BuildImageWithPackage(ctx, pkgFlags.PackagePath())
		if err != nil {
			return "", err
		}
	} else {
		keypath, err = p.BuildImage(ctx)
		if err != nil {
			return "", err
		}
	}

	err = p.CreateImage(ctx, keypath)
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "linux" {
		// linux specific config for compose - need a ifdef here
		c.RunConfig.Bridged = true  // prob. need to set the actual bridge name?
		c.RunConfig.TapName = pname // should prob. generate these - max 16 chars && no '/'

		// TODO: pass this in as uniq bridge group if under linux
		c.RunConfig.BridgeName = brName
	}

	c.RunConfig.InstanceName = pname
	c.CloudConfig.ImageName = pname
	c.RunConfig.QMP = true

	z := p.(*onprem.OnPrem)
	pid, err := z.CreateInstancePID(ctx)
	if err != nil {
		return "", err
	}

	if c.RunConfig.ShowDebug {
		fmt.Printf("%s instance with pid %s '%s' created...\n", c.CloudConfig.Platform, pid, c.RunConfig.InstanceName)
	}

	return pid, nil
}

// spawnDNS will grab whatever native pkg exists for the platform.
// no need to set a custom one.
func (com Compose) spawnDNS(non string, brName string) (string, error) {
	c := api.NewConfig()
	c.Program = "dns"

	version := api.LocalReleaseVersion
	c.Boot = path.Join(api.GetOpsHome(), version, "boot.img")
	c.RunConfig.ImageName = path.Join(api.GetOpsHome(), "images", "dns")

	// ideally all of this should happen in one place
	if c.Kernel == "" {
		version, err := getCurrentVersion()
		if err != nil {
			return "", err
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
		return "", err
	}

	c.RunConfig.InstanceName = "dns"

	keypath, err := p.BuildImageWithPackage(ctx, pkgFlags.PackagePath())
	if err != nil {
		return "", err
	}

	fmt.Println("creating image")
	err = p.CreateImage(ctx, keypath)
	if err != nil {
		return "", err
	}

	c.CloudConfig.ImageName = "dns"

	env := map[string]string{"non": non}

	c.Env = env
	c.RunConfig.QMP = true

	if runtime.GOOS == "linux" {
		// linux specific config for compose - need a ifdef here
		c.RunConfig.Bridged = true  // prob. need to set the actual bridge name?
		c.RunConfig.TapName = "dns" // right now assumes only one compose set at a time
		c.RunConfig.BridgeName = brName

		// TODO: scan for existing networks and create one not in use.
		// maybe just uniq on wordlist..
		createBridgedNetwork(brName, "")
	}

	fmt.Println("spawning instance..")
	z := p.(*onprem.OnPrem)
	pid, err := z.CreateInstancePID(ctx)
	if err != nil {
		return "", err
	}

	if c.RunConfig.ShowDebug {
		fmt.Printf("%s instance with pid %s '%s' created...\n", c.CloudConfig.Platform, pid, c.RunConfig.InstanceName)
	}

	return pid, nil
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

	// DNSSearch is the search domains of the unikernels. It is the dns_search of a docker compose
	// file, with two differences: here it is one key for all the packages (docker has it for each
	// service), and it goes to the /etc/resolv.conf of each unikernel. With "service" a program that
	// asks for "db" is also answered for db.service, the name ops registers. Give "service" first
	// or alone: see lepton.CheckSearchDomains. As in docker, "." means no search domains. In compose it
	// replaces the SearchDomains of the config.json or package.manifest of a package.
	DNSSearch DNSSearchList `yaml:"dns_search"`
}

// DNSSearchList is a list of domains that can be written as one string or as a list, as in docker.
type DNSSearchList []string

// errDNSSearchType is what a dns_search that is neither a string nor a list of strings gives.
var errDNSSearchType = errors.New("dns_search must be a string or a list of strings")

// UnmarshalYAML reads a string or a list of strings.
func (d *DNSSearchList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var one string
	if err := unmarshal(&one); err == nil {
		*d = DNSSearchList{one}
		return nil
	}
	var many []string
	if err := unmarshal(&many); err != nil {
		return errDNSSearchType
	}
	*d = many
	return nil
}

// parseCompose reads a compose file. It returns the error of yaml for a file yaml cannot read, and
// an error for a dns_search of the wrong type or with search domains that cannot go in resolv.conf.
func parseCompose(body []byte) (ComposeFile, error) {
	y := ComposeFile{}
	if err := yaml.Unmarshal(body, &y); err != nil {
		return y, err
	}
	// as in docker, "." means no search domains
	if len(y.DNSSearch) == 1 && y.DNSSearch[0] == "." {
		y.DNSSearch = nil
	}
	if err := api.CheckSearchDomains(y.DNSSearch); err != nil {
		return y, fmt.Errorf("dns_search: %v", err)
	}
	return y, nil
}

// setResolver sets the name server and the search domains the unikernel gets in /etc/resolv.conf.
func setResolver(c *types.Config, dnsIP string, search []string) {
	c.NameServers = []string{dnsIP}
	c.SearchDomains = search
}

func genNon(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length+2)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}

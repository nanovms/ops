package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "nanospkg"}
	rootCmd.AddCommand(BuildCommand())
	rootCmd.Execute()
}

type PackageManifest struct {
	Program string
	Args    []string
	Version string
	Env     map[string]string
}

func BuildCommand() *cobra.Command {
	var config string
	var version string

	var cmdBuild = &cobra.Command{
		Use:   "build [binaryname] -c [configfile] -v [version]",
		Short: "Build package from provided binary file",
		Args:  cobra.MinimumNArgs(1),
		Run:   runCmdBuild,
	}

	cmdBuild.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdBuild.PersistentFlags().StringVarP(&version, "version", "v", "", "program version")
	cmdBuild.MarkPersistentFlagRequired("version")

	return cmdBuild
}

func runCmdBuild(command *cobra.Command, args []string) {
	config, err := command.Flags().GetString("config")
	if err != nil {
		panic(err)
	}
	config = strings.TrimSpace(config)
	// fmt.Printf("%+v\n", c)

	version, err := command.Flags().GetString("version")
	if err != nil {
		panic(err)
	}

	c := cmd.UnWrapConfig(config)

	c.Program = args[0]
	progName := path.Base(c.Program)

	m, err := lepton.BuildManifest(c)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("%+v\n", m)

	pkgName := progName + "_" + version

	// Prepare package dir
	pkgDir := pkgName
	err = os.RemoveAll(pkgDir)
	if err != nil {
		panic(err)
	}
	err = os.Mkdir(pkgDir, 0770)
	if err != nil {
		panic(err)
	}

	// Generate package manifest
	var pm = PackageManifest{}
	pm.Program = c.Program
	pm.Args = m.Args()
	pm.Version = version
	pm.Env = c.Env
	// fmt.Printf("%+v\n", pm)

	// Generate package manifest JSON
	pmJson, err := json.MarshalIndent(pm, "", "  ")
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(pmJson))

	// Write package manifest
	pmPath := path.Join(pkgDir, "package.manifest")
	err = ioutil.WriteFile(pmPath, pmJson, 0660)
	if err != nil {
		panic(err)
	}

	// Create sysroot
	sysroot := path.Join(pkgDir, "sysroot")
	err = os.Mkdir(sysroot, 0770)
	if err != nil {
		panic(err)
	}

	// Populate sysroot
	files := m.Children()
	// fmt.Printf("%+v\n", files)
	err = populateSysroot(sysroot, files)
	if err != nil {
		panic(err)
	}

	// Copy README.md
	readme := "README.md"
	if _, err := os.Stat(readme); err == nil {
		Copy(readme, path.Join(pkgDir, readme))
	}

	// Pack
	archive := pkgName + ".tar.gz"
	err = TarGz(pkgDir, archive)
	if err != nil {
		panic(err)
	}
}

func populateSysroot(basePath string, files map[string]interface{}) error {
	for name, v := range files {
		curPath := path.Join(basePath, name)

		switch t := v.(type) {
		case string:
			if v == "" {
				continue
			}
			err := Copy(t, curPath)
			if err != nil {
				return err
			}
			// fmt.Printf("file: %s -> %s\n", curPath, t)
		case map[string]interface{}:
			err := os.Mkdir(curPath, 0770)
			if err != nil {
				return err
			}
			err = populateSysroot(curPath, t)
			if err != nil {
				return err
			}
			// fmt.Printf("dir: %s -> %+v\n", name, t)
		default:
			return fmt.Errorf("Unknown value %v", v)
		}
	}

	return nil
}

func Copy(src, dst string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	os.Chmod(dst, stat.Mode())
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func TarGz(src, dst string) error {
	// TODO: native golang implementation
	cpCmd := exec.Command("tar", "czf", dst, src)
	err := cpCmd.Run()
	return err
}

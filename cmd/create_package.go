package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

func execCmd(str string) {
	cmd := exec.Command("/bin/bash", "-c", str)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(out))
	}
}

func clonePackage(old string, new string, version string, newconfig *types.Config) {
	fmt.Println("cloning old pkg to new")
	o := path.Join(lepton.GetOpsHome(), "packages", old)
	n := path.Join(localPackageDirectoryPath(), new+"_"+version)

	cmd := exec.Command("mkdir", "-p", n)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(out))
	}

	str := "cp -R " + o + "/* " + n + "/"

	execCmd(str)

	ppath := n + "/package.manifest"

	c := &types.Config{}
	unWarpConfig(ppath, c)
	p := strings.Split(c.Program, "/")
	c.Program = new + "_" + version + "/" + p[1]
	c.Version = version

	addToPackage(newconfig, n)

	// nil out Dirs, Files, MapDirs or anything else we already resolved
	newconfig.Dirs = []string{}
	newconfig.MapDirs = map[string]string{}
	newconfig.Files = []string{}

	// iterate through fields and copy over anything that is not nil value
	// things like env vars need to be appended
	for i := 0; i < len(newconfig.Args); i++ {
		c.Args = append(c.Args, newconfig.Args[i])
	}

	json, _ := json.MarshalIndent(c, "", "  ")

	// would be nice to write only needed config not all config
	err = os.WriteFile(ppath, json, 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func addToPackage(newconfig *types.Config, newpath string) {
	// add any directories/files
	// {dirs, mapdirs, files}

	// add config that might be present in our directory
	// we typically use mkfs to build an image directly from config versus building
	// a package with the files
	if len(newconfig.Dirs) > 0 {
		for i := 0; i < len(newconfig.Dirs); i++ {
			str := "cp -R " + newconfig.Dirs[i] + " " + newpath + "/sysroot/."
			execCmd(str)
		}
	}

	if len(newconfig.Files) > 0 {
		for i := 0; i < len(newconfig.Files); i++ {
			str := "cp -R " + newconfig.Files[i] + " " + newpath + "/sysroot/."
			execCmd(str)
		}
	}

	if len(newconfig.MapDirs) > 0 {
		for k, v := range newconfig.MapDirs {
			newDir := newpath + "/sysroot" + v

			str := "mkdir -p " + newDir + " && cp -R " + k + " " + newDir + "/."
			execCmd(str)
		}
	}
}

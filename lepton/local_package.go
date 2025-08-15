package lepton

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nanovms/ops/log"
)

// GetLocalPackageList provides list of local packages
//
// ex valid locations:
//
// arm64/mypackage_v0.1
// arm64/gcr.io/distroless/python3-debian12_debug
func GetLocalPackageList() ([]Package, error) {
	packages := []Package{}
	var uname = GetLocalUsername()

	arches := []string{"amd64", "arm64"}

	for i := 0; i < len(arches); i++ {

		localPackagesDir := LocalPackagesRoot + "/" + arches[i]

		localPackages, err := os.ReadDir(localPackagesDir)
		if err != nil {
			log.Infof("could not find local packages directory for arch: %s\n", arches[i])
			continue
		}

		// rewrite this...
		for _, pkg := range localPackages {
			pkgName := pkg.Name()

			if !strings.Contains(pkgName, "tar.gz") {

				if !strings.Contains(pkgName, "_") {
					regPkgs, err := os.ReadDir(localPackagesDir + "/" + pkgName)
					if err != nil {
						fmt.Println(err)
					}

					packages = append(packages, findRegRooted(regPkgs, pkgName, localPackagesDir, uname)...)

				} else {

					pkg, err := emitLocalPkg(pkgName, localPackagesDir, uname)
					if err != nil {
						fmt.Println(err)
					} else {
						packages = append(packages, pkg)
					}
				}

			}
		}
	}

	return packages, nil
}

func emitLocalPkg(pkgName string, lpdir string, ns string) (Package, error) {
	var pkg Package

	herr := fmt.Sprintf("having trouble parsing the manifest of package: %s - can you verify the package.manifest is correct via jsonlint.com?\n", pkgName)

	_, name, _ := GetNSPkgnameAndVersion(pkgName)
	manifestLoc := fmt.Sprintf("%s/%s/package.manifest", lpdir, pkgName)
	if _, err := os.Stat(manifestLoc); err == nil {
		data, err := os.ReadFile(manifestLoc)
		if err != nil {
			fmt.Printf("%s\n", herr)
			os.Exit(1)
		}

		var pkg Package
		err = json.Unmarshal(data, &pkg)
		if err != nil {
			fmt.Printf("%s\n", herr)
			os.Exit(1)
		}
		pkg.Namespace = ns
		pkg.Name = name

		return pkg, nil
	}

	return pkg, errors.New("bad pkg")
}

func findRegRooted(regPkgs []os.DirEntry, pkgName string, localPackagesDir string, uname string) []Package {
	packages := []Package{}

	for x := 0; x < len(regPkgs); x++ {

		if !strings.Contains(regPkgs[x].Name(), "_") {
			rpath := localPackagesDir + "/" + pkgName + "/" + regPkgs[x].Name()

			regPkgs2, err := os.ReadDir(rpath)
			if err != nil {
				fmt.Println(err)
			}

			ns := regPkgs[x].Name()
			for y := 0; y < len(regPkgs2); y++ {

				pkg, err := emitLocalPkg(regPkgs2[y].Name(), rpath, ns)
				if err != nil {
					fmt.Println(err)
				} else {
					packages = append(packages, pkg)
				}

			}
		} else {

			pkg, err := emitLocalPkg(regPkgs[x].Name(), localPackagesDir+"/"+pkgName, uname)
			if err != nil {
				fmt.Println(err)
			} else {
				packages = append(packages, pkg)
			}

		}
	}

	return packages
}

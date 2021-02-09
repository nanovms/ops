package cmd

import (
	"fmt"
	"os"
	"path"

	api "github.com/nanovms/ops/lepton"
)

func downloadAndExtractPackage(pkg string) string {
	packagePath := api.GetOpsHome() + "/local_packages/" + pkg

	if _, err := os.Stat(packagePath); err == nil {
		return packagePath
	}

	localPackagesPath := path.Join(api.GetOpsHome(), "local_packages")
	err := os.MkdirAll(localPackagesPath, 0755)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	expackage := path.Join(localPackagesPath, pkg)
	opsPackage, err := api.DownloadPackage(pkg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	api.ExtractPackage(opsPackage, localPackagesPath)

	return expackage
}

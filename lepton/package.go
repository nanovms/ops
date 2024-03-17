package lepton

import (
	"archive/tar"
	"crypto/sha256"
	"os/exec"
	"regexp"

	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/constants"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

// PackageSysRootFolderName is the name of package root folder
const PackageSysRootFolderName = "sysroot"

var packageRegex = regexp.MustCompile(`(?P<packageName>[A-Za-z-]+)[_-](?P<version>\S+)`)

// PackageList contains a list of known packages.
type PackageList struct {
	Version  int       `json:"Version"`
	Packages []Package `json:"Packages"`
}

// Package is the definition of an OPS package.
type Package struct {
	Runtime     string `json:"runtime"`
	Version     string `json:"version"`
	Language    string `json:"language"`
	Description string `json:"description,omitempty"`
	SHA256      string `json:"sha256"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Arch        string `json:"arch"`
}

// PackageIdentifier is used to identify a namespaced package
type PackageIdentifier struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

// Match matches a package with all the fields of this identifier
func (pkgidf *PackageIdentifier) Match(pkg Package) bool {
	return pkg.Name == pkgidf.Name && pkg.Namespace == pkgidf.Namespace && pkg.Version == pkgidf.Version
}

// List returns the package list
func (pkglist *PackageList) List() []Package {
	return pkglist.Packages
}

// ParseIdentifier parses a package identifier which looks like <namespace>/<pkg>:<version>
func ParseIdentifier(identifier string) PackageIdentifier {
	tokens := strings.Split(identifier, "/")
	var namespace string
	if len(tokens) < 2 {
		namespace = ""
	} else {
		namespace = tokens[len(tokens)-2]
	}
	pkgTokens := strings.Split(tokens[len(tokens)-1], ":")
	pkgName := pkgTokens[0]
	version := "latest"
	if len(pkgTokens) > 1 {
		version = pkgTokens[1]
	}
	return PackageIdentifier{
		Name:      pkgName,
		Namespace: namespace,
		Version:   version,
	}
}

// DownloadPackage downloads package by identifier
func DownloadPackage(identifier string, config *types.Config) (string, error) {
	pkgIdf := ParseIdentifier(identifier)

	pkg, err := GetPackageMetadata(pkgIdf.Namespace, pkgIdf.Name, pkgIdf.Version)
	if err != nil {
		return "", err
	}

	if pkg == nil {
		return "", fmt.Errorf("package %q does not exist", identifier)
	}

	archivename := pkg.Namespace + "/" + pkg.Name + "_" + pkg.Version + ".tar.gz"

	archiveFolder := path.Join(PackagesCache, pkg.Namespace)
	os.MkdirAll(archiveFolder, 0755)
	packagepath := path.Join(PackagesCache, archivename)
	_, err = os.Stat(packagepath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err == nil {
		return packagepath, nil
	}

	archivePath := pkg.Namespace + "/" + pkg.Name + "/" + pkg.Version + ".tar.gz"

	pkgBaseURL := PackageBaseURL

	// Check config override
	if config != nil {
		cPkgBaseURL := strings.Trim(config.PackageBaseURL, " ")
		if len(cPkgBaseURL) > 0 {
			pkgBaseURL = cPkgBaseURL
		}
	}

	// Check environment variable override
	ePkgBaseURL := os.Getenv("OPS_PACKAGE_BASE_URL")
	if ePkgBaseURL != "" {
		pkgBaseURL = ePkgBaseURL
	}

	isNetworkRepo := !strings.HasPrefix(pkgBaseURL, "file://")
	if isNetworkRepo {
		var fileURL string
		if strings.HasSuffix(pkgBaseURL, "/") {
			fileURL = pkgBaseURL + archivePath
		} else {
			fileURL = fmt.Sprintf("%s/%s", pkgBaseURL, archivePath)
		}

		if AltGOARCH != "" {
			if AltGOARCH == "arm64" {
				fileURL = strings.ReplaceAll(fileURL, ".tar.gz", "/arm64.tar.gz")
			}

		} else {
			if runtime.GOARCH == "arm64" {
				fileURL = strings.ReplaceAll(fileURL, ".tar.gz", "/arm64.tar.gz")
			}
		}

		if err = DownloadFileWithProgress(packagepath, fileURL, 600); err != nil {
			return "", err
		}

		return packagepath, nil
	}

	pkgBaseURL = strings.TrimPrefix(pkgBaseURL, "file://")
	srcPath := filepath.Join(pkgBaseURL, archivename)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	srcStat, err := srcFile.Stat()
	if err != nil {
		return "", err
	}

	destFile, err := os.Create(packagepath)
	if err != nil {
		return "", err
	}
	defer destFile.Close()

	progressCounter := NewWriteCounter(int(srcStat.Size()))
	progressCounter.Start()
	_, err = io.Copy(destFile, io.TeeReader(srcFile, progressCounter))
	if err != nil {
		return "", err
	}
	progressCounter.Finish()

	return packagepath, nil
}

// GetPackageList provides list of packages
func GetPackageList(config *types.Config) (*PackageList, error) {
	var err error

	pkgManifestURL := PackageManifestURL

	// Check config override
	if config != nil {
		cPkgManifestURL := strings.Trim(config.PackageManifestURL, " ")
		if len(cPkgManifestURL) > 0 {
			pkgManifestURL = cPkgManifestURL
		}
	}

	// Check environment var override
	ePkgManifestURL := os.Getenv("OPS_PACKAGE_MANIFEST_URL")
	if ePkgManifestURL != "" {
		pkgManifestURL = ePkgManifestURL
	}

	packageManifest := GetPackageManifestFile()
	if strings.HasPrefix(pkgManifestURL, "file://") {
		destFile, err := os.OpenFile(packageManifest, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
		if err != nil {
			return nil, err
		}
		defer destFile.Close()

		pkgManifestURL = strings.TrimPrefix(pkgManifestURL, "file://")
		srcFile, err := os.Open(pkgManifestURL)
		if err != nil {
			return nil, err
		}
		defer srcFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			return nil, err
		}
	} else {
		stat, err := os.Stat(packageManifest)
		if os.IsNotExist(err) || PackageManifestChanged(stat, pkgManifestURL) {
			if err = DownloadFile(packageManifest, pkgManifestURL, 10, false); err != nil {
				return nil, err
			}
		}
	}

	var packages PackageList
	data, err := os.ReadFile(packageManifest)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &packages)
	if err != nil {
		return nil, err
	}

	return &packages, nil
}

// GetLocalPackageList provides list of local packages
func GetLocalPackageList() ([]Package, error) {
	packages := []Package{}

	localPackagesDir := GetOpsHome() + "/local_packages"

	localPackages, err := os.ReadDir(localPackagesDir)
	if err != nil {
		return nil, err
	}
	username := GetLocalUsername()
	for _, pkg := range localPackages {
		pkgName := pkg.Name()

		// ignore packages compressed
		if !strings.Contains(pkgName, "tar.gz") {
			_, name, _ := GetNSPkgnameAndVersion(pkgName)
			manifestLoc := fmt.Sprintf("%s/%s/package.manifest", localPackagesDir, pkgName)
			if _, err := os.Stat(manifestLoc); err == nil {

				data, err := os.ReadFile(manifestLoc)
				if err != nil {
					fmt.Printf("having trouble parsing the manifest of package: %s - can you verify the package.manifest is correct via jsonlint.com?\n", pkgName)
					os.Exit(1)
					return nil, err
				}

				var pkg Package
				err = json.Unmarshal(data, &pkg)
				if err != nil {
					fmt.Printf("having trouble parsing the manifest of package: %s - can you verify the package.manifest is correct via jsonlint.com?\n", pkgName)
					os.Exit(1)
					return nil, err
				}
				pkg.Namespace = username
				pkg.Name = name
				packages = append(packages, pkg)
			}

		}
	}

	return packages, nil
}

func getPackageCache() string {
	packagefolder := path.Join(GetOpsHome(), "packages")
	if _, err := os.Stat(packagefolder); os.IsNotExist(err) {
		os.MkdirAll(packagefolder, 0755)
	}
	return packagefolder
}

// GetPackageManifestFile give path for package manifest file
func GetPackageManifestFile() string {
	return path.Join(getPackageCache(), PackageManifestFileName)
}

// PackageManifestChanged verifies if package manifest changed
func PackageManifestChanged(fino os.FileInfo, remoteURL string) bool {
	res, err := http.Head(remoteURL)
	if err != nil {
		var netError *net.Error
		if errors.Is(err, *netError) {
			fmt.Printf(constants.WarningColor, "missing internet?, using local manifest.\n")
		} else {
			fmt.Printf("probably bad URL: %s, got error %s", remoteURL, err)
			os.Exit(1)
		}

		return false
	}

	return fino.Size() != res.ContentLength
}

func sha256Of(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// ExtractPackage extracts package in ops home.
// This function is currently over-loaded.
func ExtractPackage(archive, dest string, config *types.Config) {
	sha := sha256Of(archive)
	homeDirName := filepath.Base(GetOpsHome())

	// hack
	// this only verifies for packages - unfortunately this function is
	// used for extracting releases (which currently don't have
	// checksums)
	if strings.Contains(archive, filepath.Join(homeDirName, "packages")) {
		fname := filepath.Base(archive)
		namespace := filepath.Base(filepath.Dir(archive))
		fname = strings.ReplaceAll(fname, ".tar.gz", "")
		fnameTokens := strings.Split(fname, "_")
		pkgName := fnameTokens[0]
		version := fnameTokens[len(fnameTokens)-1]

		pkg, _ := GetPackageMetadata(namespace, pkgName, version)
		if pkg == nil || pkg.SHA256 != sha {
			log.Fatalf("This package doesn't match what is in the manifest.")
		}

	}

	in, err := os.Open(archive)
	if err != nil {
		fmt.Printf("File missing: %s", archive)
		os.Exit(1)
	}
	gzip, err := gzip.NewReader(in)
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}
	defer gzip.Close()
	tr := tar.NewReader(gzip)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Printf(err.Error())
			os.Exit(1)
		}
		if header == nil {
			continue
		}
		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					fmt.Printf("Failed to create directory %s, error is: %s", target, err)
					os.Exit(1)
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				fmt.Printf("Failed open file %s, error is %s", target, err)
				os.Exit(1)
			}
			if err := f.Truncate(0); err != nil {
				fmt.Printf("Failed truncate file %s, error is %s", target, err)
				os.Exit(1)
			}
			if _, err := io.Copy(f, tr); err != nil {
				fmt.Printf("Failed tar file %s, error is %s", target, err.Error())
				os.Exit(1)
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil {
				log.Warn(err.Error())
			}
		}
	}
}

// GetNSPkgnameAndVersion gets the namespace, name and version from the pkg identifier
func GetNSPkgnameAndVersion(pkgIdentifier string) (string, string, string) {
	namespace, pkgIdf := ExtractNS(pkgIdentifier)
	match := packageRegex.FindStringSubmatch(pkgIdf)
	result := make(map[string]string)
	// mostly then there is no version in the name and hence we can return all of it
	if len(match) == 0 {
		return namespace, pkgIdf, "latest"
	}
	for i, name := range packageRegex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	return namespace, result["packageName"], result["version"]
}

// ExtractNS extracts namespace from the package identifier of format <namespace>/<packageWithVersion>
// and returns the namespace and package with version
func ExtractNS(identifier string) (string, string) {
	namespace := ""
	pkgIdfs := strings.Split(identifier, "/")
	pkgIdf := pkgIdfs[0]
	if len(pkgIdfs) > 1 {
		namespace = pkgIdfs[0]
		pkgIdf = pkgIdfs[1]
	}
	return namespace, pkgIdf
}

func localPackageDirectoryPath() string {
	return path.Join(GetOpsHome(), "local_packages")
}

// ClonePackage will cloned a package from ~/.ops/packages to
// ~/.ops/local_packages.
func ClonePackage(old string, newPkg string, version string, oldconfig *types.Config, newconfig *types.Config) {
	fmt.Println("cloning old pkg to new")
	o := path.Join(GetOpsHome(), "packages", old)
	n := path.Join(localPackageDirectoryPath(), newPkg+"_"+version)

	execCmd("mkdir -p", n, "&&", "cp -R", o+"/*", n+"/")

	ppath := n + "/package.manifest"

	c := oldconfig
	p := strings.Split(c.Program, "/")
	c.Program = newPkg + "_" + version + "/" + p[1]
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
	err := os.WriteFile(ppath, json, 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

// CreatePackageFromRun builds a new package as if you were doing an
// 'ops run myprogram'
func CreatePackageFromRun(newPkg string, version string, mergedCfg *types.Config) {
	fmt.Println("creating new pkg")
	pkgDir := filepath.Join(localPackageDirectoryPath(), newPkg+"_"+version)
	pkgSysrootDir := filepath.Join(pkgDir, PackageSysRootFolderName)

	execCmd("mkdir -p", pkgDir)
	execCmd("mkdir -p", pkgSysrootDir)

	// package - manifest
	pkgCfg := newManifestConfig(mergedCfg)

	// customize
	pkgCfg.Args = []string{mergedCfg.Program}
	pkgCfg.Program = newPkg + "_" + version + "/" + mergedCfg.Program
	pkgCfg.Version = version

	json, err := json.MarshalIndent(pkgCfg, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	err = os.WriteFile(filepath.Join(pkgDir, "package.manifest"), json, 0666)
	if err != nil {
		fmt.Println(err)
	}

	// package - program
	execCmd("cp", mergedCfg.Program, pkgDir+"/.")

	// package - {dirs, mapdirs, files} - config won't be present in manifest
	addToPackage(mergedCfg, pkgDir)

	// package - shared libs
	deps, err := getSharedLibs("", mergedCfg.Program, nil)
	if err != nil {
		fmt.Println(err)
	}
	// create dir layout + cp files
	for k, v := range deps {
		execCmd("mkdir -p", filepath.Join(pkgSysrootDir, filepath.Dir(v)))
		execCmd("cp", v, filepath.Join(pkgSysrootDir, k))
	}

}

func addToPackage(newconfig *types.Config, newpath string) {
	// add any directories/files
	// {dirs, mapdirs, files}

	// add config that might be present in our directory
	// we typically use mkfs to build an image directly from config versus building
	// a package with the files

	if len(newconfig.Dirs) == 0 && len(newconfig.Files) == 0 && len(newconfig.MapDirs) == 0 {
		return
	}

	pkgSysrootDir := filepath.Join(newpath, PackageSysRootFolderName)
	execCmd("mkdir -p", pkgSysrootDir)

	for i := 0; i < len(newconfig.Dirs); i++ {
		execCmd("cp -R", newconfig.Dirs[i], pkgSysrootDir+"/.")
	}
	for i := 0; i < len(newconfig.Files); i++ {
		execCmd("cp -R", newconfig.Files[i], pkgSysrootDir+"/.")
	}
	for k, v := range newconfig.MapDirs {
		newDir := filepath.Join(pkgSysrootDir, v)
		execCmd("mkdir -p", newDir, "&&", "cp -R", k, newDir+"/.")
	}
}

func execCmd(str ...string) {
	cmd := exec.Command("/bin/bash", "-c", strings.Join(str, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(out))
	}
}

// NewManifestConfig - from a merged config, select configs for manifest file,
//
//	contains shallow copies
func newManifestConfig(c *types.Config) *types.Config {
	return &types.Config{
		Program:             c.Program,             //
		Args:                c.Args,                // shallow copy
		BaseVolumeSz:        c.BaseVolumeSz,        //
		Debugflags:          c.Debugflags,          // shallow copy
		Env:                 c.Env,                 // shallow copy
		ManifestPassthrough: c.ManifestPassthrough, // shallow copy
		RebootOnExit:        c.RebootOnExit,        //
		RunConfig: types.RunConfig{
			CanIPForward: c.RunConfig.CanIPForward, // gcp
			GPUs:         c.RunConfig.GPUs,         // gcp
			GPUType:      c.RunConfig.GPUType,      // gcp
			Ports:        c.RunConfig.Ports,        // shallow copy
			UDPPorts:     c.RunConfig.UDPPorts,     // shallow copy
		},
		Klibs: c.Klibs, // shallow copy

		Version:     c.Version,
		Language:    c.Language,
		Runtime:     c.Runtime,
		Description: c.Description,

		// default zero value

		// NoTrace:                   c.NoTrace,                   // shallow copy
		// Dirs:                      c.Dirs,                      // shallow copy
		// Files:                     c.Files,                     // shallow copy
		// DisableArgsCopy:           c.DisableArgsCopy,           //
		// Boot:                      c.Boot,                      //
		// UefiBoot:                  c.UefiBoot,                  //
		// Uefi:                      c.Uefi,                      //
		// BuildDir:                  c.BuildDir,                  //
		// CloudConfig:               c.CloudConfig,               // shallow copy
		// TargetConfig:              c.TargetConfig,              // shallow copy
		// Force:                     c.Force,                     //
		// Kernel:                    c.Kernel,                    //
		// KlibDir:                   c.KlibDir,                   //
		// MapDirs:                   c.MapDirs,                   // shallow copy
		// Mounts:                    c.Mounts,                    // shallow copy
		// NameServers:               c.NameServers,               // shallow copy
		// NanosVersion:              c.NanosVersion,              //
		// NightlyBuild:              c.NightlyBuild,              //
		// ProgramPath:               c.ProgramPath,               //
		// LocalFilesParentDirectory: c.LocalFilesParentDirectory, //
		// TargetRoot:                c.TargetRoot,                //
		// VolumesDir:                c.VolumesDir,                //
		// PackageBaseURL:            c.PackageBaseURL,            //
		// PackageManifestURL:        c.PackageManifestURL,        //
	}
}

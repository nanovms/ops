package fs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/nanovms/ops/log"
)

// link refers to a link filetype
type link struct {
	path string
}

// ManifestNetworkConfig has network configuration to set static IP
type ManifestNetworkConfig struct {
	IP      string
	IPv6    string
	Gateway string
	NetMask string
}

// Manifest represent the filesystem.
type Manifest struct {
	root        map[string]interface{} // root fs
	boot        map[string]interface{} // boot fs
	targetRoot  string
	klibHostDir string
}

// NewManifest init
func NewManifest(targetRoot string) *Manifest {
	m := &Manifest{
		root:       mkFS(),
		targetRoot: targetRoot,
	}
	m.root["arguments"] = make([]string, 0)
	m.root["environment"] = make(map[string]interface{})
	return m
}

// AddNetworkConfig adds network configuration
func (m *Manifest) AddNetworkConfig(networkConfig *ManifestNetworkConfig) {
	m.root["ipaddr"] = networkConfig.IP
	m.root["netmask"] = networkConfig.NetMask
	m.root["gateway"] = networkConfig.Gateway
	m.root["ip6addr"] = networkConfig.IPv6
}

// AddUserProgram adds user program
func (m *Manifest) AddUserProgram(imgpath string) (err error) {
	parts := strings.Split(imgpath, "/")
	if parts[0] == "." {
		parts = parts[1:]
	}
	program := path.Join("/", path.Join(parts...))

	err = m.AddFile(program, imgpath)
	if err != nil {
		return
	}

	m.SetProgram(program)

	return
}

// SetProgram sets user program
func (m *Manifest) SetProgram(program string) {
	m.root["program"] = program
}

// SetKlibDir sets the host directory where kernel libs are located
func (m *Manifest) SetKlibDir(dir string) {
	m.klibHostDir = dir
}

// AddMount adds mount
func (m *Manifest) AddMount(label, path string) {
	dir := strings.TrimPrefix(path, "/")
	mkDirPath(m.rootDir(), dir)
	if m.root["mounts"] == nil {
		m.root["mounts"] = make(map[string]interface{})
	}
	mounts := m.root["mounts"].(map[string]interface{})
	mounts[label] = path
}

// AddEnvironmentVariable adds environment variables
func (m *Manifest) AddEnvironmentVariable(name string, value string) {
	env := m.root["environment"].(map[string]interface{})
	env[name] = value
}

// AddKlibs append klibs to manifest file if they don't exist
func (m *Manifest) AddKlibs(klibs []string) {
	if len(klibs) == 0 {
		return
	}
	if m.boot == nil {
		m.boot = mkFS()
	}
	klibDir := mkDir(m.bootDir(), "klib")
	hostDir := m.klibHostDir
	for _, klib := range klibs {
		klibPath := hostDir + "/" + klib
		if _, err := os.Stat(klibPath); !os.IsNotExist(err) {
			m.AddFileTo(klibDir, klib, klibPath)
		} else {
			fmt.Printf("Klib %s not found in directory %s\n", klib, hostDir)
		}
	}
	m.root["klibs"] = "bootfs"
}

// AddArgument add commandline arguments to
// user program
func (m *Manifest) AddArgument(arg string) {
	args := m.root["arguments"].([]string)
	m.root["arguments"] = append(args, arg)
}

// AddDebugFlag enables debug flags
func (m *Manifest) AddDebugFlag(name string, value rune) {
	m.root[name] = string(value)
}

// AddNoTrace enables debug flags
func (m *Manifest) AddNoTrace(name string) {
	if m.root["notrace"] == nil {
		m.root["notrace"] = make([]string, 0)
	}
	notrace := m.root["notrace"].([]string)
	m.root["notrace"] = append(notrace, name)
}

// AddKernel the kernel to use
func (m *Manifest) AddKernel(path string) {
	if m.boot == nil {
		m.boot = mkFS()
	}
	m.AddFileTo(m.bootDir(), "kernel", path)
}

// AddDirectory adds all files in dir to image
func (m *Manifest) AddDirectory(dir string, workDir string) error {
	if err := os.Chdir(workDir); err != nil {
		return err
	}

	err := filepath.Walk(dir, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// if the path is relative then root it to image path
		var vmpath string
		if hostpath[0] != '/' {
			vmpath = "/" + hostpath
		} else {
			vmpath = hostpath
		}

		if (info.Mode() & os.ModeSymlink) != 0 {
			info, err = os.Stat(hostpath)
			if err != nil {
				fmt.Printf("warning: %v\n", err)
				// ignore invalid symlinks
				return nil
			}

			// add link and continue on
			err = m.AddLink(vmpath, hostpath)
			if err != nil {
				return err
			}

			return nil
		}

		if info.IsDir() {
			parts := strings.FieldsFunc(vmpath, func(c rune) bool { return c == '/' })
			node := m.rootDir()
			for i := 0; i < len(parts); i++ {
				if _, ok := node[parts[i]]; !ok {
					node[parts[i]] = make(map[string]interface{})
				}
				if reflect.TypeOf(node[parts[i]]).Kind() == reflect.String {
					err := fmt.Errorf("directory %s is conflicting with an existing file", hostpath)
					log.Error(err)
					return err
				}
				node = node[parts[i]].(map[string]interface{})
			}
		} else {
			err = m.AddFile(vmpath, hostpath)
			if err != nil {
				return err
			}
		}
		return nil

	})
	return err
}

// AddRelativeDirectory adds all files in dir to image
func (m *Manifest) AddRelativeDirectory(src string) error {
	err := filepath.Walk(src, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		vmpath := "/" + strings.TrimPrefix(hostpath, src)

		if (info.Mode() & os.ModeSymlink) != 0 {
			info, err = os.Stat(hostpath)
			if err != nil {
				fmt.Printf("warning: %v\n", err)
				// ignore invalid symlinks
				return nil
			}

			// add link and continue on
			err = m.AddLink(vmpath, hostpath)
			if err != nil {
				return err
			}

			return nil
		}

		if info.IsDir() {
			parts := strings.FieldsFunc(vmpath, func(c rune) bool { return c == '/' })
			node := m.rootDir()
			for i := 0; i < len(parts); i++ {
				if _, ok := node[parts[i]]; !ok {
					node[parts[i]] = make(map[string]interface{})
				}
				if reflect.TypeOf(node[parts[i]]).Kind() == reflect.String {
					err := fmt.Errorf("directory %s is conflicting with an existing file", hostpath)
					log.Error(err)
					return err
				}
				node = node[parts[i]].(map[string]interface{})
			}
		} else {
			err = m.AddFile(vmpath, hostpath)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// FileExists checks if file is present at path in manifest
func (m *Manifest) FileExists(filepath string) bool {
	parts := strings.FieldsFunc(filepath, func(c rune) bool { return c == '/' })
	node := m.rootDir()
	for i := 0; i < len(parts)-1; i++ {
		if _, ok := node[parts[i]]; !ok {
			return false
		}
		node = node[parts[i]].(map[string]interface{})
	}
	pathtest := node[parts[len(parts)-1]]
	if pathtest != nil && reflect.TypeOf(pathtest).Kind() == reflect.String {
		return true
	}
	return false
}

// AddLink to add a file to manifest
func (m *Manifest) AddLink(filepath string, hostpath string) error {
	parts := strings.FieldsFunc(filepath, func(c rune) bool { return c == '/' })
	node := m.rootDir()

	for i := 0; i < len(parts)-1; i++ {
		if _, ok := node[parts[i]]; !ok {
			node[parts[i]] = make(map[string]interface{})
		}
		node = node[parts[i]].(map[string]interface{})
	}

	pathtest := node[parts[len(parts)-1]]
	if pathtest != nil && reflect.TypeOf(pathtest).Kind() != reflect.String {
		err := fmt.Errorf("file %s overriding an existing directory", filepath)
		log.Error(err)
		return err
	}

	if pathtest != nil && reflect.TypeOf(pathtest).Kind() == reflect.String && node[parts[len(parts)-1]] != hostpath {
		fmt.Printf("warning: overwriting existing file %s hostpath old: %s new: %s\n", filepath, node[parts[len(parts)-1]], hostpath)
	}

	_, err := LookupFile(m.targetRoot, hostpath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file \"%s\" is missing: %v", hostpath, err)
		}
		return err
	}

	s, err := os.Readlink(hostpath)
	if err != nil {
		log.Fatalf("bad link")
	}

	node[parts[len(parts)-1]] = link{path: s}
	return nil
}

// AddFile to add a file to manifest
func (m *Manifest) AddFile(filepath string, hostpath string) error {
	return m.AddFileTo(m.rootDir(), filepath, hostpath)
}

// AddFileTo adds a file to a given directory
func (m *Manifest) AddFileTo(dir map[string]interface{}, filepath string, hostpath string) error {
	parts := strings.FieldsFunc(filepath, func(c rune) bool { return c == '/' })
	node := dir

	for i := 0; i < len(parts)-1; i++ {
		if _, ok := node[parts[i]]; !ok {
			node[parts[i]] = make(map[string]interface{})
		}
		node = node[parts[i]].(map[string]interface{})
	}

	pathtest := node[parts[len(parts)-1]]
	if pathtest != nil && reflect.TypeOf(pathtest).Kind() != reflect.String {
		err := fmt.Errorf("file '%s' overriding an existing directory", filepath)
		log.Fatal(err)
	}

	if pathtest != nil && reflect.TypeOf(pathtest).Kind() == reflect.String && pathtest != hostpath {
		fmt.Printf("warning: overwriting existing file %s hostpath old: %s new: %s\n", filepath, pathtest, hostpath)
	}

	_, err := LookupFile(m.targetRoot, hostpath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file \"%s\" is missing: %v", hostpath, err)
		}
		return err
	}

	node[parts[len(parts)-1]] = hostpath
	return nil
}

// AddLibrary to add a dependent library
func (m *Manifest) AddLibrary(path string) {
	parts := strings.FieldsFunc(path, func(c rune) bool { return c == '/' })
	node := m.rootDir()
	for i := 0; i < len(parts)-1; i++ {
		node = mkDir(node, parts[i])
	}
	m.AddFileTo(node, parts[len(parts)-1], path)
}

// AddPassthrough to add key, value directly to manifest
func (m *Manifest) AddPassthrough(key, value string) {
	m.root[key] = value
}

func (m *Manifest) finalize() {
	if m.boot != nil {
		klibDir, isDir := m.bootDir()["klib"].(map[string]interface{})
		if isDir && (klibDir["ntp"] != nil) {
			env := m.root["environment"].(map[string]interface{})
			var err error

			var ntpAddress string
			var ntpPort string
			var ntpPollMin string
			var ntpPollMax string
			var ntpResetThreshold string

			var pollMinNumber int
			var pollMaxNumber int

			if val, ok := env["ntpAddress"].(string); ok {
				ntpAddress = val
			}

			if val, ok := env["ntpPort"].(string); ok {
				ntpPort = val
			}

			if val, ok := env["ntpPollMin"].(string); ok {
				pollMinNumber, err = strconv.Atoi(val)
				if err == nil && pollMinNumber > 3 {
					ntpPollMin = val
				}
			}

			if val, ok := env["ntpPollMax"].(string); ok {
				pollMaxNumber, err = strconv.Atoi(val)
				if err == nil && pollMaxNumber < 18 {
					ntpPollMax = val
				}
			}

			if val, ok := env["ntpResetThreshold"].(string); ok {
				_, err = strconv.Atoi(val)
				if err == nil {
					ntpResetThreshold = val
				}
			}

			if pollMinNumber != 0 && pollMaxNumber != 0 && pollMinNumber > pollMaxNumber {
				ntpPollMin = ""
				ntpPollMax = ""
			}

			if ntpAddress != "" {
				m.root["ntp_address"] = ntpAddress
			}

			if ntpPort != "" {
				m.root["ntp_port"] = ntpPort
			}

			if ntpPollMin != "" {
				m.root["ntp_poll_min"] = ntpPollMin
			}

			if ntpPollMax != "" {
				m.root["ntp_poll_max"] = ntpPollMax
			}

			if ntpResetThreshold != "" {
				m.root["ntp_reset_threshold"] = ntpResetThreshold
			}
		}
	}
}

func (m *Manifest) bootDir() map[string]interface{} {
	return getRootDir(m.boot)
}

func (m *Manifest) rootDir() map[string]interface{} {
	return getRootDir(m.root)
}

// LookupFile look up file path in target root directory
func LookupFile(targetRoot string, path string) (string, error) {
	if targetRoot != "" {
		var targetPath string
		currentPath := path
		for {
			targetPath = filepath.Join(targetRoot, currentPath)
			fi, err := os.Lstat(targetPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return path, err
				}
				// lookup on host
				break
			}

			if fi.Mode()&os.ModeSymlink == 0 {
				// not a symlink found in target root
				return targetPath, nil
			}

			currentPath, err = os.Readlink(targetPath)
			if err != nil {
				return path, err
			}

			if currentPath[0] != '/' {
				// relative symlinks are ok
				path = targetPath
				break
			}

			// absolute symlinks need to be resolved again
		}
	}

	_, err := os.Stat(path)

	return path, err
}

func mkDir(parent map[string]interface{}, dir string) map[string]interface{} {
	subDir := parent[dir]
	if subDir != nil {
		return subDir.(map[string]interface{})
	}
	newDir := make(map[string]interface{})
	parent[dir] = newDir
	return newDir
}

// MkdirPath is mkDirPath() using root directory as parent
func (m *Manifest) MkdirPath(path string) {
	mkDirPath(m.rootDir(), path)
}

func mkDirPath(parent map[string]interface{}, path string) map[string]interface{} {
	parts := strings.Split(path, "/")
	for _, element := range parts {
		parent = mkDir(parent, element)
	}
	return parent
}

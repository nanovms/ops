package lepton

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

// Manifest represent the filesystem.
type Manifest struct {
	sb          strings.Builder
	children    map[string]interface{}
	program     string
	args        []string
	debugFlags  map[string]rune
	noTrace     []string
	environment map[string]string
	targetRoot  string
}

// NewManifest init
func NewManifest(targetRoot string) *Manifest {
	return &Manifest{
		children:    make(map[string]interface{}),
		debugFlags:  make(map[string]rune),
		environment: make(map[string]string),
		targetRoot:  targetRoot,
	}
}

// AddUserProgram adds user program
func (m *Manifest) AddUserProgram(imgpath string) {

	parts := strings.Split(imgpath, "/")
	if parts[0] == "." {
		parts = parts[1:]
	}
	m.program = path.Join("/", path.Join(parts...))
	err := m.AddFile(m.program, imgpath)
	if err != nil {
		panic(err)
	}
}

// AddEnvironmentVariable adds envirnoment variables
func (m *Manifest) AddEnvironmentVariable(name string, value string) {
	m.environment[name] = value
}

// AddArgument add commandline arguments to
// user program
func (m *Manifest) AddArgument(arg string) {
	m.args = append(m.args, arg)
}

// AddDebugFlag enables debug flags
func (m *Manifest) AddDebugFlag(name string, value rune) {
	m.debugFlags[name] = value
}

// AddDebugFlag enables debug flags
func (m *Manifest) AddNoTrace(name string) {
	m.noTrace = append(m.noTrace, name)
}

// AddKernel the kernel to use
func (m *Manifest) AddKernel(path string) {
	m.children["kernel"] = path
}

// AddRelative path
func (m *Manifest) AddRelative(key string, path string) {
	m.children[key] = path
}

// AddDirectory adds all files in dir to image
func (m *Manifest) AddDirectory(dir string) error {
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
		}
		if info.IsDir() {
			parts := strings.FieldsFunc(vmpath, func(c rune) bool { return c == '/' })
			node := m.children
			for i := 0; i < len(parts); i++ {
				if _, ok := node[parts[i]]; !ok {
					node[parts[i]] = make(map[string]interface{})
				}
				if reflect.TypeOf(node[parts[i]]).Kind() == reflect.String {
					err := fmt.Errorf("directory %s is conflicting with an existing file", hostpath)
					fmt.Println(err)
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
	node := m.children
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

// AddFile to add a file to manifest
func (m *Manifest) AddFile(filepath string, hostpath string) error {
	parts := strings.FieldsFunc(filepath, func(c rune) bool { return c == '/' })
	node := m.children
	for i := 0; i < len(parts)-1; i++ {
		if _, ok := node[parts[i]]; !ok {
			node[parts[i]] = make(map[string]interface{})
		}
		node = node[parts[i]].(map[string]interface{})
	}
	pathtest := node[parts[len(parts)-1]]
	if pathtest != nil && reflect.TypeOf(pathtest).Kind() != reflect.String {
		err := fmt.Errorf("file %s overriding an existing directory", filepath)
		fmt.Println(err)
		return err
	}
	if pathtest != nil && reflect.TypeOf(pathtest).Kind() == reflect.String && node[parts[len(parts)-1]] != hostpath {
		fmt.Printf("warning: overwriting existing file %s hostpath old: %s new: %s\n", filepath, node[parts[len(parts)-1]], hostpath)
	}
	_, err := lookupFile(m.targetRoot, hostpath)
	if err != nil {
		return err
	}
	node[parts[len(parts)-1]] = hostpath
	return nil
}

// AddLibrary to add a dependent library
func (m *Manifest) AddLibrary(path string) {
	parts := strings.FieldsFunc(path, func(c rune) bool { return c == '/' })
	node := m.children
	for i := 0; i < len(parts)-1; i++ {
		if _, ok := node[parts[i]]; !ok {
			node[parts[i]] = make(map[string]interface{})
		}
		node = node[parts[i]].(map[string]interface{})
	}
	node[parts[len(parts)-1]] = path
}

// AddUserData adds all files in dir to
// final image.
func (m *Manifest) AddUserData(dir string) {
	// TODO
}

func escapeValue(s string) string {
	if strings.Contains(s, "\"") {
		s = strings.Replace(s, "\"", "\\\"", -1)
	}
	if strings.ContainsAny(s, "\":()[] \t\n") {
		s = "\"" + s + "\""
	}
	return s
}

func (m *Manifest) String() string {
	sb := m.sb
	sb.WriteString("(\n")
	sb.WriteString("children:(\n")
	toString(&m.children, &sb, 4)
	sb.WriteString(")\n")

	//program
	sb.WriteString("program:")
	sb.WriteString(m.program)
	sb.WriteRune('\n')

	// arguments
	sb.WriteString("arguments:[")
	if len(m.args) > 0 {
		fmt.Println(m.args)
		escapedArgs := make([]string, len(m.args))
		for i, arg := range m.args {
			escapedArgs[i] = escapeValue(arg)
		}
		sb.WriteString(strings.Join(escapedArgs, " "))
	}
	sb.WriteString("]\n")

	// debug
	for k, v := range m.debugFlags {
		sb.WriteString(k)
		sb.WriteRune(':')
		sb.WriteRune(v)
		sb.WriteRune('\n')
	}

	// notrace
	if len(m.noTrace) > 0 {
		sb.WriteString("notrace:[")
		sb.WriteString(strings.Join(m.noTrace, " "))
		sb.WriteString("]\n")
	}

	// environment
	n := len(m.environment)
	sb.WriteString("environment:(")
	for k, v := range m.environment {
		n = n - 1
		sb.WriteString(k)
		sb.WriteRune(':')
		sb.WriteString(escapeValue(v))
		if n > 0 {
			sb.WriteRune(' ')
		}
	}
	sb.WriteString(")\n")

	//
	sb.WriteString(")\n")
	return sb.String()
}

func toString(m *map[string]interface{}, sb *strings.Builder, indent int) {
	for k, v := range *m {
		value, ok := v.(string)
		sb.WriteString(strings.Repeat(" ", indent))
		if ok {
			sb.WriteString(escapeValue(k))
			sb.WriteString(":(contents:(host:")
			sb.WriteString(escapeValue(value))
			sb.WriteString("))\n")
		} else {
			sb.WriteString(k)
			sb.WriteString(":(children:(")
			// recur
			ch := v.(map[string]interface{})
			if len(ch) > 0 {
				sb.WriteRune('\n')
				toString(&ch, sb, indent+4)
				sb.WriteString(strings.Repeat(" ", indent))
			}
			sb.WriteString("))\n")
		}
	}
}

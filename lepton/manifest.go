package lepton

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Manifest represent the filesystem.
type Manifest struct {
	sb          strings.Builder
	children    map[string]interface{}
	program     string
	args        []string
	debugFlags  map[string]rune
	environment map[string]string
}

// NewManifest init
func NewManifest() *Manifest {
	return &Manifest{
		children:    make(map[string]interface{}),
		debugFlags:  make(map[string]rune),
		environment: make(map[string]string),
	}
}

// AddUserProgram adds user program
func (m *Manifest) AddUserProgram(imgpath string) {

	parts := strings.Split(imgpath, "/")
	if parts[0] == "." {
		parts = parts[1:]
	}
	m.program = path.Join("/", path.Join(parts...))
	m.AddFile(m.program, imgpath)
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

		if info.IsDir() {
			parts := strings.FieldsFunc(vmpath, func(c rune) bool { return c == '/' })
			node := m.children
			for i := 0; i < len(parts); i++ {
				if _, ok := node[parts[i]]; !ok {
					node[parts[i]] = make(map[string]interface{})
				}
				node = node[parts[i]].(map[string]interface{})
			}
		} else {
			m.AddFile(vmpath, hostpath)
		}
		return nil
	})
	return err
}

// AddFile to add a file to manifest
func (m *Manifest) AddFile(filepath string, hostpath string) {
	parts := strings.FieldsFunc(filepath, func(c rune) bool { return c == '/' })
	node := m.children
	for i := 0; i < len(parts)-1; i++ {
		if _, ok := node[parts[i]]; !ok {
			node[parts[i]] = make(map[string]interface{})
		}
		node = node[parts[i]].(map[string]interface{})
	}
	node[parts[len(parts)-1]] = hostpath
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

func (m *Manifest) String() string {
	sb := m.sb
	sb.WriteRune('(')
	sb.WriteString("children:(")
	toString(&m.children, &sb)
	sb.WriteRune(')')

	//program
	sb.WriteRune('\n')
	sb.WriteString("program:")
	sb.WriteString(m.program)
	sb.WriteRune('\n')

	// arguments
	sb.WriteString("arguments:[")
	if len(m.args) > 0 {
		fmt.Println(m.args)
		sb.WriteString(strings.Join(m.args, " "))
	}
	sb.WriteRune(']')
	sb.WriteRune('\n')

	// debug
	for k, v := range m.debugFlags {
		sb.WriteString(k)
		sb.WriteRune(':')
		sb.WriteRune(v)
		sb.WriteRune('\n')
	}

	// envirnoment
	n := len(m.environment)
	sb.WriteString("environment:(")
	for k, v := range m.environment {
		n = n - 1
		sb.WriteString(k)
		sb.WriteRune(':')
		sb.WriteString(v)
		if n > 0 {
			sb.WriteRune(' ')
		}
	}
	//
	sb.WriteRune(')')
	sb.WriteRune(')')
	return sb.String()
}

func toString(m *map[string]interface{}, sb *strings.Builder) {
	for k, v := range *m {
		value, ok := v.(string)
		if ok {
			sb.WriteString(k)
			sb.WriteRune(':')
			sb.WriteRune('(')
			sb.WriteString("contents:(host:")
			sb.WriteString(value)
			sb.WriteRune(')')
			sb.WriteRune(')')
		} else {
			sb.WriteString(k)
			sb.WriteRune(':')
			sb.WriteRune('(')
			sb.WriteString("children:(")
			// recur
			ch := v.(map[string]interface{})
			toString(&ch, sb)
			sb.WriteRune(')')
			sb.WriteRune(')')
		}
	}
}

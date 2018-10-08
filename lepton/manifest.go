package lepton

import (
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
	var elfname = filepath.Base(imgpath)
	var extension = filepath.Ext(elfname)
	elfname = elfname[0 : len(elfname)-len(extension)]
	m.children[elfname] = imgpath
	m.program = path.Join("/", elfname)
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

// AddKernal the kernel to use
func (m *Manifest) AddKernal(path string) {
	m.children["kernel"] = path
}

// AddRelative path
func (m *Manifest) AddRelative(key string, path string) {
	m.children[key] = path
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
	_, filename := filepath.Split(m.program)
	sb.WriteString(filename)
	if len(m.args) > 0 {
		sb.WriteRune(' ')
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

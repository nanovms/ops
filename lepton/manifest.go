package lepton

import (
	"path/filepath"
	"strings"
)

// Node represent either a leaf node or a parent node.
type Node struct {
	name     string
	path     string
	children []*Node
}

// Manifest represent the filesystem.
type Manifest struct {
	sb          strings.Builder
	root        *Node
	program     string
	args        []string
	debugFlags  map[string]rune
	environment map[string]string
}

func (n *Node) addContentNode(key string, hostpath string) {
	newnode := Node{name: key, path: hostpath}
	n.children = append(n.children, &newnode)
}

// Create a new file manifest
func NewManifest() *Manifest {
	return &Manifest{
		root:        &Node{},
		debugFlags:  make(map[string]rune),
		environment: make(map[string]string),
	}
}

// User program path
func (m *Manifest) SetProgramPath(path string) {
	m.program = path
}

// Any envirnoment variables that need to be set
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

// AddKernel adds kernel to root node.
func (m *Manifest) AddKernel(path string) {
	m.root.addContentNode("kernel", path)
}

// AddLib adds dependent lib to manifest
func (m *Manifest) AddLib(path string, recur bool) {
	parts := strings.Split(path, "/")
	node := m.root
	if recur {
		// construct the path
		for _, p := range parts[:len(parts)-1] {
			newnode := &Node{name: p}
			node.children = append(node.children, newnode)
			node = newnode
		}
	}
	// add the content node as leaf
	node.addContentNode(parts[len(parts)-1], path)
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
	// children goes here
	for _, c := range m.root.children {
		sb.WriteString(c.String())
		sb.WriteRune('\n')
	}
	sb.WriteRune(')')

	//program
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

func nodeToString(n *Node, sb *strings.Builder) {
	if len(n.children) == 0 {
		sb.WriteString(n.name)
		sb.WriteRune(':')
		sb.WriteRune('(')
		sb.WriteString("contents:(host:")
		sb.WriteString(n.path)
		sb.WriteRune(')')
		sb.WriteRune(')')
	} else {
		sb.WriteString(n.name)
		sb.WriteRune(':')
		sb.WriteRune('(')
		sb.WriteString("children:")
		sb.WriteRune('(')
		for _, c := range n.children {
			nodeToString(c, sb)
		}
		sb.WriteRune(')')
		sb.WriteRune(')')
	}
}

func (n *Node) String() string {
	var sb strings.Builder
	nodeToString(n, &sb)
	return sb.String()
}

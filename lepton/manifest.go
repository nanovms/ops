package lepton

import (
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
	node        *Node
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
	return &Manifest{node: &Node{}}
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
	m.node.addContentNode("kernel", path)
}

// AddUserLibrary adss a user library and construct the path
// if recur is true.
func (m *Manifest) AddUserLibrary(path string, recur bool) {
	parts := strings.Split(path, "/")
	node := m.node
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

func (m *Manifest) String() string {
	// TODO : Serialize the manifest for
	// mkfs.
	return ""
}

package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func addDummyKlib(m *Manifest, name string) {
	m.boot = mkFS()
	klibDir := mkDir(m.bootDir(), "klib")
	klibDir[name] = "dummy"
}

func TestManifestWithArgs(t *testing.T) {
	m := NewManifest("")
	m.AddArgument("/bin/ls")
	m.AddArgument("first")
	args := m.root["arguments"].([]string)
	assert.Equal(t, 2, len(args))
	assert.Equal(t, "/bin/ls", args[0])
	assert.Equal(t, "first", args[1])
}

func TestManifestWithEnv(t *testing.T) {
	m := NewManifest("")
	m.AddEnvironmentVariable("var1", "value1")
	env := m.root["environment"].(map[string]interface{})
	assert.Equal(t, "value1", env["var1"])
}

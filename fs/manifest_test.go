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

func TestAddNTPEnvVarsToManifestFile(t *testing.T) {

	t.Run("should not add ntp manifest variables if ntp klib is not specified", func(t *testing.T) {
		m := NewManifest("")
		m.AddEnvironmentVariable("ntpAddress", "127.0.0.1")
		m.AddEnvironmentVariable("ntpPort", "1234")
		m.AddEnvironmentVariable("ntpPollMin", "5")
		m.AddEnvironmentVariable("ntpPollMax", "10")
		m.finalize()

		assert.Equal(t, nil, m.root["ntp_address"])
		assert.Equal(t, nil, m.root["ntp_port"])
		assert.Equal(t, nil, m.root["ntp_poll_min"])
		assert.Equal(t, nil, m.root["ntp_poll_max"])
	})

	t.Run("should add ntp manifest variables if ntp klib is added and environment variables are valid", func(t *testing.T) {
		m := NewManifest("")
		addDummyKlib(m, "ntp")
		m.AddEnvironmentVariable("ntpAddress", "127.0.0.1")
		m.AddEnvironmentVariable("ntpPort", "1234")
		m.AddEnvironmentVariable("ntpPollMin", "5")
		m.AddEnvironmentVariable("ntpPollMax", "10")
		m.finalize()

		assert.Equal(t, "127.0.0.1", m.root["ntp_address"])
		assert.Equal(t, "1234", m.root["ntp_port"])
		assert.Equal(t, "5", m.root["ntp_poll_min"])
		assert.Equal(t, "10", m.root["ntp_poll_max"])
	})

	t.Run("should not ntp poll limits if min is greater than max", func(t *testing.T) {
		m := NewManifest("")
		addDummyKlib(m, "ntp")
		m.AddEnvironmentVariable("ntpPollMin", "10")
		m.AddEnvironmentVariable("ntpPollMax", "5")
		m.finalize()

		assert.Equal(t, nil, m.root["ntp_poll_min"])
		assert.Equal(t, nil, m.root["ntp_poll_max"])
	})

	t.Run("should not add ntp poll min value if is lower than 4", func(t *testing.T) {
		m := NewManifest("")
		addDummyKlib(m, "ntp")
		m.AddEnvironmentVariable("ntpPollMin", "3")
		m.finalize()

		assert.Equal(t, nil, m.root["ntp_poll_min"])
	})

	t.Run("should add ntp poll min value if is greater than 3", func(t *testing.T) {
		m := NewManifest("")
		addDummyKlib(m, "ntp")
		m.AddEnvironmentVariable("ntpPollMin", "5")
		m.finalize()

		assert.Equal(t, "5", m.root["ntp_poll_min"])
	})

	t.Run("should not add ntp poll max value if is greater than 17", func(t *testing.T) {
		m := NewManifest("")
		addDummyKlib(m, "ntp")
		m.AddEnvironmentVariable("ntpPollMax", "18")
		m.finalize()

		assert.Equal(t, nil, m.root["ntp_poll_max"])
	})

	t.Run("should add ntp poll max value if is lower than 18", func(t *testing.T) {
		m := NewManifest("")
		addDummyKlib(m, "ntp")
		m.AddEnvironmentVariable("ntpPollMax", "17")
		m.finalize()

		assert.Equal(t, "17", m.root["ntp_poll_max"])
	})

	t.Run("should not add ntp poll max/min values if they are not numbers", func(t *testing.T) {
		m := NewManifest("")
		addDummyKlib(m, "ntp")
		m.AddEnvironmentVariable("ntpPollMin", "asd")
		m.AddEnvironmentVariable("ntpPollMax", "dsa")
		m.finalize()

		assert.Equal(t, nil, m.root["ntp_poll_min"])
		assert.Equal(t, nil, m.root["ntp_poll_max"])
	})

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

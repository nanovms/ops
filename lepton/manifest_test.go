package lepton

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddKernel(t *testing.T) {
	m := NewManifest("")
	kernel := getLastReleaseLocalFolder() + "/kernel.img"
	m.AddKernel(kernel)
	s := m.bootDir()["kernel"]
	if s != kernel {
		t.Errorf("Expected:%v Actual:%v", kernel, s)
	}
}

func TestManifestWithDeps(t *testing.T) {
	var c Config
	c.Kernel = getLastReleaseLocalFolder() + "/kernel.img"
	c.Program = "../data/main"
	c.TargetRoot = os.Getenv("NANOS_TARGET_ROOT")
	m, err := BuildManifest(&c)
	if err != nil {
		t.Fatal(err)
	}
	m.AddDirectory("../data/static")
}

func TestAddKlibs(t *testing.T) {

	t.Run("should add klibs to manifest", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"tls", "cloud_init"})
		klibDir := m.bootDir()["klib"].(map[string]interface{})

		got := klibDir["tls"]
		want := getKlibsDir(false) + "/tls"

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		got = klibDir["cloud_init"]
		want = getKlibsDir(false) + "/cloud_init"

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("should add klibs to manifest if they do not exist yet", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"tls", "cloud_init"})
		m.AddKlibs([]string{"tls", "radar"})

		klibDir := m.bootDir()["klib"].(map[string]interface{})
		got := len(klibDir)
		want := len([]string{"tls", "cloud_init", "radar"})

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestAddRadarEnvAddKlibs(t *testing.T) {
	m := NewManifest("")
	m.AddEnvironmentVariable("RADAR_KEY", "TEST")
	klibDir := m.bootDir()["klib"].(map[string]interface{})

	got := klibDir["tls"]
	want := getKlibsDir(false) + "/tls"

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	got = klibDir["radar"]
	want = getKlibsDir(false) + "/radar"

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
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
		m.AddKlibs([]string{"ntp"})
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
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMin", "10")
		m.AddEnvironmentVariable("ntpPollMax", "5")
		m.finalize()

		assert.Equal(t, nil, m.root["ntp_poll_min"])
		assert.Equal(t, nil, m.root["ntp_poll_max"])
	})

	t.Run("should not add ntp poll min value if is lower than 4", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMin", "3")
		m.finalize()

		assert.Equal(t, nil, m.root["ntp_poll_min"])
	})

	t.Run("should add ntp poll min value if is greater than 3", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMin", "5")
		m.finalize()

		assert.Equal(t, "5", m.root["ntp_poll_min"])
	})

	t.Run("should not add ntp poll max value if is greater than 17", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMax", "18")
		m.finalize()

		assert.Equal(t, nil, m.root["ntp_poll_max"])
	})

	t.Run("should add ntp poll max value if is lower than 18", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMax", "17")
		m.finalize()

		assert.Equal(t, "17", m.root["ntp_poll_max"])
	})

	t.Run("should not add ntp poll max/min values if they are not numbers", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
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

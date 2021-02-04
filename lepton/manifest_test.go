package lepton

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	relpath = `hw:(contents:(host:examples/hw))
`
	kernel = `kernel:(contents:(host:kernel/kernel))
`
	lib = `lib:(children:(
    x86_64-linux-gnu:(children:(
        libc.so.6:(contents:(host:/lib/x86_64-linux-gnu/libc.so.6))
    ))
))
`
)

func TestAddKernel(t *testing.T) {
	m := NewManifest("")
	m.AddKernel("kernel/kernel")
	var sb strings.Builder
	toString(&m.boot, &sb, 0)
	s := sb.String()
	if s != kernel {
		t.Errorf("Expected:%v Actual:%v", kernel, s)
	}
}

func TestAddRelativePath(t *testing.T) {
	m := NewManifest("")
	m.AddRelative("hw", "examples/hw")
	var sb strings.Builder
	toString(&m.children, &sb, 0)
	s := sb.String()
	if s != relpath {
		t.Errorf("Expected:%v Actual:%v", relpath, s)
	}
}

func TestAddLibs(t *testing.T) {
	m := NewManifest("")
	m.AddLibrary("/lib/x86_64-linux-gnu/libc.so.6")
	var sb strings.Builder
	toString(&m.children, &sb, 0)
	s := sb.String()
	if s != lib {
		t.Errorf("Expected:%v Actual:%v", lib, s)
	}
}

func TestManifestWithDeps(t *testing.T) {
	var c Config
	c.Program = "../data/main"
	c.TargetRoot = os.Getenv("NANOS_TARGET_ROOT")
	m, err := BuildManifest(&c)
	if err != nil {
		t.Fatal(err)
	}
	m.AddDirectory("../data/static")
}

func TestSerializeManifest(t *testing.T) {
	m := NewManifest("")
	m.AddUserProgram("/bin/ls")
	m.AddKernel("kernel/kernel")
	m.AddArgument("first")
	m.AddEnvironmentVariable("var1", "value1")
	m.AddLibrary("/usr/local/u.so")
	m.AddLibrary("/usr/local/two.so")
	s := m.String()
	// this is bogus
	if len(s) < 100 {
		t.Errorf("Unexpected")
	}
}

func TestAddKlibs(t *testing.T) {

	t.Run("should add klibs to manifest", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"tls", "cloud_init"})

		got := m.klibs
		want := []string{"tls", "cloud_init"}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("should add klibs to manifest if they do not exist yet", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"tls", "cloud_init"})
		m.AddKlibs([]string{"tls", "radar"})

		got := m.klibs
		want := []string{"tls", "cloud_init", "radar"}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestAddRadarEnvAddKlibs(t *testing.T) {
	m := NewManifest("")
	m.AddEnvironmentVariable("RADAR_KEY", "TEST")

	got := m.klibs
	want := []string{"tls", "radar"}

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

		manifestFile := m.String()

		assert.NotContains(t, manifestFile, "ntp_address:127.0.0.1\n")
		assert.NotContains(t, manifestFile, "ntp_port:1234\n")
		assert.NotContains(t, manifestFile, "ntp_poll_min:5\n")
		assert.NotContains(t, manifestFile, "ntp_poll_max:10\n")
	})

	t.Run("should add ntp manifest variables if ntp klib is added and environment variables are valid", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpAddress", "127.0.0.1")
		m.AddEnvironmentVariable("ntpPort", "1234")
		m.AddEnvironmentVariable("ntpPollMin", "5")
		m.AddEnvironmentVariable("ntpPollMax", "10")

		manifestFile := m.String()

		assert.Contains(t, manifestFile, "ntp_address:127.0.0.1\n")
		assert.Contains(t, manifestFile, "ntp_port:1234\n")
		assert.Contains(t, manifestFile, "ntp_poll_min:5\n")
		assert.Contains(t, manifestFile, "ntp_poll_max:10\n")
	})

	t.Run("should not ntp poll limits if min is greater than max", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMin", "10")
		m.AddEnvironmentVariable("ntpPollMax", "5")

		manifestFile := m.String()

		assert.NotContains(t, manifestFile, "ntp_poll_min:10\n")
		assert.NotContains(t, manifestFile, "ntp_poll_max:5\n")
	})

	t.Run("should not add ntp poll min value if is lower than 4", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMin", "3")

		manifestFile := m.String()

		assert.NotContains(t, manifestFile, "ntp_poll_min:5\n")
	})

	t.Run("should add ntp poll min value if is greater than 3", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMin", "5")

		manifestFile := m.String()

		assert.Contains(t, manifestFile, "ntp_poll_min:5\n")
	})

	t.Run("should not add ntp poll max value if is greater than 17", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMax", "18")

		manifestFile := m.String()

		assert.NotContains(t, manifestFile, "ntp_poll_max:18\n")
	})

	t.Run("should add ntp poll max value if is lower than 18", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMax", "17")

		manifestFile := m.String()

		assert.Contains(t, manifestFile, "ntp_poll_max:17\n")
	})

	t.Run("should not add ntp poll max/min values if they are not numbers", func(t *testing.T) {
		m := NewManifest("")
		m.AddKlibs([]string{"ntp"})
		m.AddEnvironmentVariable("ntpPollMin", "asd")
		m.AddEnvironmentVariable("ntpPollMax", "dsa")

		manifestFile := m.String()

		assert.NotContains(t, manifestFile, "ntp_poll_min:asd\n")
		assert.NotContains(t, manifestFile, "ntp_poll_max:dsa\n")
	})

}

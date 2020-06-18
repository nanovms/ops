package lepton

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	relpath = `hw:(contents:(host:examples/hw))
`
	kernel = `kernel:(contents:(host:stage3/stage3))
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
	m.AddKernel("stage3/stage3")
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
	fmt.Println(m.String())
}

func TestSerializeManifest(t *testing.T) {
	m := NewManifest("")
	m.AddUserProgram("/bin/ls")
	m.AddKernel("stage3/stage3")
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

package lepton

import (
	"strings"
	"testing"
)

const (
	relpath = `hw:(contents:(host:examples/hw))`
	kernal  = `kernel:(contents:(host:state3/stage3))`
	lib     = `lib:(children:(x86_64-linux-gnu:(children:` +
		`(libc.so.6:(contents:(host:/lib/x86_64-linux-gnu/libc.so.6))id-2.23.so:` +
		`(contents:(host:/lib/x86_64-linux-gnu/id-2.23.so))))))`
)

func TestAddKernal(t *testing.T) {
	m := NewManifest()
	m.AddKernal("state3/stage3")
	var sb strings.Builder
	toString(&m.children, &sb)
	s := sb.String()
	if s != kernal {
		t.Errorf("Expected:%v Actual:%v", kernal, s)
	}
}

func TestAddRelativePath(t *testing.T) {
	m := NewManifest()
	m.AddRelative("hw", "examples/hw")
	var sb strings.Builder
	toString(&m.children, &sb)
	s := sb.String()
	if s != relpath {
		t.Errorf("Expected:%v Actual:%v", relpath, s)
	}
}

func TestAddLibs(t *testing.T) {
	m := NewManifest()
	m.AddLibrary("/lib/x86_64-linux-gnu/libc.so.6")
	m.AddLibrary("/lib/x86_64-linux-gnu/id-2.23.so")
	var sb strings.Builder
	toString(&m.children, &sb)
	s := sb.String()
	if s != lib {
		t.Errorf("Expected:%v Actual:%v", lib, s)
	}
}

func TestManifestWithDeps(t *testing.T) {
	_, err := buildManifest("/home/tijoytom/nanovms/nanos/examples/webg")
	if err != nil {
		t.Fatal(err)
	}
	// TODO : verification
}

func TestSerializeManifest(t *testing.T) {
	m := NewManifest()
	m.AddUserProgram("/hws")
	m.AddKernal("stage3/stage3")
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

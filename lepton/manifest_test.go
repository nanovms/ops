package lepton

import (
	"fmt"
	"sort"
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

type runeSorter []rune

func (s runeSorter) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s runeSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s runeSorter) Len() int {
	return len(s)
}

func sortString(s string) string {
	r := []rune(s)
	sort.Sort(runeSorter(r))
	return string(r)
}

func TestAddLibs(t *testing.T) {
	m := NewManifest()
	m.AddLibrary("/lib/x86_64-linux-gnu/libc.so.6")
	m.AddLibrary("/lib/x86_64-linux-gnu/id-2.23.so")
	var sb strings.Builder
	toString(&m.children, &sb)
	s := sortString(sb.String())
	if s != sortString(lib) {
		t.Errorf("Expected:%v Actual:%v", lib, s)
	}
}

func TestManifestWithDeps(t *testing.T) {
	var c Config
	initDefaultImages(&c)
	c.Program = "../data/main"
	m, err := BuildManifest(&c)
	if err != nil {
		t.Fatal(err)
	}
	m.AddDirectory("../data/static")
	fmt.Println(m.String())
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

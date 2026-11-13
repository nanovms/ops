package lepton

import (
	"strings"
	"testing"

	"github.com/nanovms/ops/fs"
	"github.com/nanovms/ops/types"
)

func TestResolvConfHasTheSearchLineOnlyWhenThereAreDomains(t *testing.T) {
	tests := []struct {
		name string
		c    types.Config
		want string
	}{
		{"name servers only", types.Config{NameServers: []string{"8.8.8.8"}}, "nameserver  8.8.8.8\n"},
		{"with a search domain", types.Config{NameServers: []string{"192.168.33.2"}, SearchDomains: []string{"service"}}, "nameserver  192.168.33.2\nsearch service\n"},
		{"two domains", types.Config{NameServers: []string{"1.1.1.1", "8.8.8.8"}, SearchDomains: []string{"a", "b"}}, "nameserver  1.1.1.1\nnameserver  8.8.8.8\nsearch a b\n"},
		{"nothing", types.Config{}, ""},
	}
	for _, tt := range tests {
		got, err := resolvConf(&tt.c)
		if err != nil || string(got) != tt.want {
			t.Errorf("%s: %q, %v, want %q", tt.name, got, err, tt.want)
		}
	}
}

// SearchDomains is a key of config.json, so the check is on the way to resolv.conf and not only
// in ops compose
func TestResolvConfRefusesSearchDomainsThatInjectALine(t *testing.T) {
	for _, d := range []string{"a\nnameserver 6.6.6.6", "a b", "a\tb", "a\rb"} {
		c := types.Config{NameServers: []string{"8.8.8.8"}, SearchDomains: []string{d}}
		got, err := resolvConf(&c)
		if err == nil {
			t.Errorf("%q gave %q", d, got)
		} else if !strings.Contains(err.Error(), "SearchDomains") {
			t.Errorf("the error does not name the key: %v", err)
		}
	}
}

func TestCheckSearchDomains(t *testing.T) {
	long := strings.Repeat("a", 63)
	for _, ok := range [][]string{nil, {"service"}, {"a.b", "c"}, {"service."}, {"my_net", "a-b.c1"}, {long + "." + long}, {"a", "b", "c", "d", "e", "f"}} {
		if err := CheckSearchDomains(ok); err != nil {
			t.Errorf("%v: %v", ok, err)
		}
	}
	bad := []string{"", ".", "a..b", ".a", "-a", "a b", "a\tb", "a\rb", "a\nb", "a-", "a.b-", "a;b", "a/b", "#x", "a#b", "ü.com",
		long + "a", strings.Repeat(long+".", 4)}
	for _, b := range bad {
		if err := CheckSearchDomains([]string{b}); err == nil {
			t.Errorf("%q was accepted", b)
		}
	}
	if err := CheckSearchDomains([]string{"a", "b", "c", "d", "e", "f", "g"}); err == nil {
		t.Error("seven domains were accepted")
	}
	// the line is "search ", the domains with a space between, and a newline: musl ignores a line
	// of 256 bytes
	dom := func(n int) string { return strings.Repeat("a", n) }
	if err := CheckSearchDomains([]string{dom(40), dom(40), dom(40), dom(40), dom(40), dom(43)}); err == nil {
		t.Error("a search line of 256 bytes was accepted")
	}
	if err := CheckSearchDomains([]string{dom(40), dom(40), dom(40), dom(40), dom(40), dom(42)}); err != nil {
		t.Errorf("a search line of 255 bytes: %v", err)
	}
}

// an invalid SearchDomains is an error that goes back to the caller, the library does not exit
func TestAddDNSConfigReturnsTheSearchDomainsError(t *testing.T) {
	c := types.Config{BuildDir: t.TempDir(), NameServers: []string{"8.8.8.8"}, SearchDomains: []string{"a b"}}
	err := addDNSConfig(fs.NewManifest(t.TempDir()), &c)
	if err == nil || !strings.Contains(err.Error(), "SearchDomains") {
		t.Errorf("error = %v, want one that names SearchDomains", err)
	}
	c.SearchDomains = []string{"service"}
	if err := addDNSConfig(fs.NewManifest(t.TempDir()), &c); err != nil {
		t.Errorf("valid search domains: %v", err)
	}
}

package cmd

import (
	"reflect"
	"testing"

	"github.com/nanovms/ops/types"
)

func TestParseComposeReadsDNSSearch(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"a list", "dns_search:\n  - service\npackages:\n  - pkg: db\n", []string{"service"}},
		{"a string", "dns_search: service\npackages:\n  - pkg: db\n", []string{"service"}},
		{"two in a list", "dns_search: [service, example.com]\n", []string{"service", "example.com"}},
		{"without the key", "packages:\n  - pkg: db\n", nil},
		{"null", "dns_search:\npackages:\n  - pkg: db\n", nil},
		{"a dot, as docker, is none", "dns_search: \".\"\n", nil},
		{"a list with a dot, is none", "dns_search: [\".\"]\n", nil},
	}
	for _, tt := range tests {
		y, err := parseCompose([]byte(tt.body))
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
		}
		if len(y.DNSSearch) != len(tt.want) || (len(tt.want) > 0 && !reflect.DeepEqual([]string(y.DNSSearch), tt.want)) {
			t.Errorf("%s: DNSSearch = %v, want %v", tt.name, y.DNSSearch, tt.want)
		}
	}
}

// the check is done on the way in, for both forms of the key
func TestParseComposeRefusesABadDNSSearch(t *testing.T) {
	for _, body := range []string{
		"dns_search:\n  - \"a\\tb\"\n",
		"dns_search:\n  - \"a\\rb\"\n",
		"dns_search: \"a b\"\n",
		"dns_search: \"a\\nnameserver 6.6.6.6\"\n",
		"dns_search: [a, b, c, d, e, f, g]\n",
		"dns_search: {a: b}\n",
		"dns_search: [[service]]\n",
		"dns_search: [a-]\n",
	} {
		if _, err := parseCompose([]byte(body)); err == nil {
			t.Errorf("%q was accepted", body)
		}
	}
}

func TestSetResolverSetsTheDNSSearch(t *testing.T) {
	var c types.Config
	setResolver(&c, "192.168.33.2", []string{"service"})
	if !reflect.DeepEqual(c.NameServers, []string{"192.168.33.2"}) || !reflect.DeepEqual(c.SearchDomains, []string{"service"}) {
		t.Errorf("%+v", c)
	}
	setResolver(&c, "192.168.33.2", nil)
	if len(c.SearchDomains) != 0 {
		t.Errorf("SearchDomains = %v, want none", c.SearchDomains)
	}
}

// UP stops on the error of parseCompose, so nothing is started from a file yaml cannot read
func TestParseComposeReturnsTheYamlError(t *testing.T) {
	for _, body := range []string{
		"packages: [unclosed\n",
		"packages:\n  - pkg: db\n bad indent: x\n",
		"packages: notalist\n",
	} {
		if _, err := parseCompose([]byte(body)); err == nil {
			t.Errorf("%q was accepted", body)
		}
	}
}

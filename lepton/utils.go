package lepton

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"unicode/utf8"
)

func exitWithError(errs string) {
	log.Fatal(errs)
}

// isDomainValid returns an error if the domain name is not valid
// See https://tools.ietf.org/html/rfc1034#section-3.5 and
// https://tools.ietf.org/html/rfc1123#section-2.
// code extracted from https://gist.github.com/chmike/d4126a3247a6d9a70922fc0e8b4f4013
func isDomainValid(name string) error {
	switch {
	case len(name) == 0:
		return fmt.Errorf("domain: empty")
	case len(name) > 255:
		return fmt.Errorf("domain: name length is %d, can't exceed 255", len(name))
	}
	var l int
	for i := 0; i < len(name); i++ {
		b := name[i]
		if b == '.' {
			// check domain labels validity
			switch {
			case i == l:
				return fmt.Errorf("domain: invalid character '%c' at offset %d: label can't begin with a period", b, i)
			case i-l > 63:
				return fmt.Errorf("domain: byte length of label '%s' is %d, can't exceed 63", name[l:i], i-l)
			case name[l] == '-':
				return fmt.Errorf("domain: label '%s' at offset %d begins with a hyphen", name[l:i], l)
			case name[i-1] == '-':
				return fmt.Errorf("domain: label '%s' at offset %d ends with a hyphen", name[l:i], l)
			}
			l = i + 1
			continue
		}
		// test label character validity, note: tests are ordered by decreasing validity frequency
		if !(b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-' || b >= 'A' && b <= 'Z') {
			// show the printable unicode character starting at byte offset i
			c, _ := utf8.DecodeRuneInString(name[i:])
			if c == utf8.RuneError {
				return fmt.Errorf("domain: invalid rune at offset %d", i)
			}
			return fmt.Errorf("domain: invalid character '%c' at offset %d", c, i)
		}
	}
	// check top level domain validity
	switch {
	case l == len(name):
		return fmt.Errorf("domain: missing top level domain, domain can't end with a period")
	case len(name)-l > 63:
		return fmt.Errorf("domain: byte length of top level domain '%s' is %d, can't exceed 63", name[l:], len(name)-l)
	case name[l] == '-':
		return fmt.Errorf("domain: top level domain '%s' at offset %d begins with a hyphen", name[l:], l)
	case name[len(name)-1] == '-':
		return fmt.Errorf("domain: top level domain '%s' at offset %d ends with a hyphen", name[l:], l)
	case name[l] >= '0' && name[l] <= '9':
		return fmt.Errorf("domain: top level domain '%s' at offset %d begins with a digit", name[l:], l)
	}
	return nil
}

// SliceAtoi converts slice of strings to a slice of integers
func SliceAtoi(sa []string) ([]int, error) {
	si := make([]int, 0, len(sa))
	for _, a := range sa {
		i, err := strconv.Atoi(a)
		if err != nil {
			return si, err
		}
		si = append(si, i)
	}
	return si, nil
}

// BaseHTTPRequest is a wrapper around http.NewRequest with a header that holds the ops version.
func BaseHTTPRequest(method string, url string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, url, body)
	if request != nil {
		request.Header.Add("X-ops-version", Version)
	}
	return request, err
}

package cmd

import (
	"strings"
	"unicode"

	"github.com/go-errors/errors"
)

// ValidateNetworkPorts verifies ports strings have right format
// Strings must have only numbers, commas or hyphens. Commas and hypens must separate 2 numbers
func ValidateNetworkPorts(ports []string) error {
	for _, str := range ports {
		var hyphenUsed bool

		if str[0] == ',' || str[len(str)-1] == ',' {
			return errors.Errorf("\"%s\" commas must separate numbers", str)
		} else if str[0] == '-' || str[len(str)-1] == '-' {
			return errors.Errorf("\"%s\" hyphen must separate two numbers", str)
		}

		for i, ch := range str {
			if ch == ',' {
				if !unicode.IsDigit(rune(str[i-1])) || !unicode.IsDigit(rune(str[i+1])) {
					return errors.Errorf("\"%s\" commas must separate numbers", str)
				}
			} else if ch == '-' {
				if hyphenUsed {
					return errors.Errorf("\"%s\" may have only one hyphen", str)
				} else if !unicode.IsDigit(rune(str[i-1])) || !unicode.IsDigit(rune(str[i+1])) {
					return errors.Errorf("\"%s\" hyphen must separate two numbers", str)
				}
				hyphenUsed = true
			} else if !unicode.IsDigit(ch) {
				return errors.Errorf("\"%s\" must have only numbers, commas or one hyphen", str)
			}
		}

	}

	return nil
}

// PrepareNetworkPorts validates ports and split ports strings separated by commas
func PrepareNetworkPorts(ports []string) (portsPrepared []string, err error) {
	err = ValidateNetworkPorts(ports)
	if err != nil {
		return
	}

	for _, ports := range ports {
		portsPrepared = append(portsPrepared, strings.Split(ports, ",")...)
	}

	return
}

package cmd

import (
	"reflect"
	"testing"
)

func TestValidateNetworkPorts(t *testing.T) {

	t.Run("should return error if ports have wrong format", func(t *testing.T) {
		tests := []struct {
			ports       []string
			valid       bool
			errExpected string
		}{
			{[]string{"80"}, true, ""},
			{[]string{"80-100"}, true, ""},
			{[]string{"80,90,100"}, true, ""},
			{[]string{"80-8080,9000"}, true, ""},
			{[]string{"9000,80-8080"}, true, ""},
			{[]string{"hello"}, false, "\"hello\" must have only numbers, commas or one hyphen"},
			{[]string{"-80"}, false, "\"-80\" hyphen must separate two numbers"},
			{[]string{"80-8080-9000"}, false, "\"80-8080-9000\" may have only one hyphen"},
			{[]string{"80,"}, false, "\"80,\" commas must separate numbers"},
		}

		for _, tt := range tests {
			err := validateNetworkPorts(tt.ports)
			if err != nil && tt.valid {
				t.Errorf("Expected %s to be valid, got next error %s", tt.ports, err.Error())
			} else if err == nil && !tt.valid {
				t.Errorf("Expected %s to be invalid", tt.ports)
			}

			if !tt.valid && err != nil && err.Error() != tt.errExpected {
				t.Errorf("expected \"%s\", got \"%s\" (%s)", tt.errExpected, err.Error(), tt.ports)
			}
		}

	})

}

func TestPrepareNetworkPorts(t *testing.T) {

	t.Run("separate ports separated by commas", func(t *testing.T) {

		got, _ := prepareNetworkPorts([]string{"80,8080", "9000-10000"})
		want := []string{"80", "8080", "9000-10000"}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

	})

}

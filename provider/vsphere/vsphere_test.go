//go:build vsphere || !onlyprovider

package vsphere

import (
	"os"
	"testing"
)

func TestVsphere_getCredentials(t *testing.T) {
	tests := []struct {
		name      string
		govcURL   string
		user      string
		pass      string
		shouldErr bool
	}{
		{
			name:      "full",
			govcURL:   "https://user:pass@host:8080",
			user:      "",
			pass:      "",
			shouldErr: false,
		},
		{
			name:      "separate",
			govcURL:   "https://@host:8080",
			user:      "user",
			pass:      "pass",
			shouldErr: false,
		},
		{
			name:      "full/no_protocol",
			govcURL:   "user:pass@host:8080",
			user:      "",
			pass:      "",
			shouldErr: false,
		},
		{
			name:      "separate/no_protocol",
			govcURL:   "host:8080",
			user:      "user",
			pass:      "pass",
			shouldErr: false,
		},
		{
			name:      "no_username",
			govcURL:   "https://:pass@host:8080",
			user:      "",
			pass:      "",
			shouldErr: true,
		},
		{
			name:      "empty_password",
			govcURL:   "https://user:@host:8080",
			user:      "",
			pass:      "",
			shouldErr: true,
		},
	}
	v := NewProvider()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("GOVC_URL", tt.govcURL)
			os.Setenv("GOVC_USERNAME", tt.user)
			os.Setenv("GOVC_PASSWORD", tt.pass)
			_, err := v.getCredentials()
			if (err != nil) != tt.shouldErr {
				t.Errorf("expected err %v, got error:\n%s", tt.shouldErr, err.Error())
			}
		})
	}
}

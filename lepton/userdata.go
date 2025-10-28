package lepton

import "encoding/base64"

// EncodeUserDataBase64 encodes user data as a base64 string.
// Returns an empty string if the input is empty.
// Used by cloud providers that require base64-encoded user data.
func EncodeUserDataBase64(userData string) string {
	if userData == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(userData))
}

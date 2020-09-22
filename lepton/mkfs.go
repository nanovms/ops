package lepton

import (
	"strings"
)

// uuidFromMKFS reads the resulting UUID from nanos mkfs tool
func uuidFromMKFS(b []byte) string {
	var uuid string
	in := string(b)
	fields := strings.Fields(in)
	for i, f := range fields {
		if strings.Contains(f, "UUID") {
			uuid = fields[i+1]
		}
	}
	return strings.TrimSpace(uuid)
}

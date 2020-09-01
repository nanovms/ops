package lepton

import (
	"fmt"
	"strings"
)

func uuidFromMKFS(b []byte) string {
	var uuid string
	in := string(b)
	fields := strings.Fields(in)
	fmt.Println(fields)
	for i, f := range fields {
		if strings.Contains(f, "UUID") {
			uuid = fields[i+1]
		}
	}
	return strings.TrimSpace(uuid)
}

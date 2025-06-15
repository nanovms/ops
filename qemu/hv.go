package qemu

import (
	"os"
)

func isInception() bool {
	if os.Getenv("INCEPTION") != "" {
		return true
	}

	return false
}

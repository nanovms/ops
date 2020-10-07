package lepton

import (
	"fmt"
	"os"
)

func exitWithError(errs string) {
	fmt.Println(fmt.Sprintf(ErrorColor, errs))
	os.Exit(1)
}

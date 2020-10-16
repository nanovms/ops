package lepton

import (
	"fmt"
	"os"
	"strings"
)

func exitWithError(errs string) {
	fmt.Println(fmt.Sprintf(ErrorColor, errs))
	os.Exit(1)
}

func arrayToString(a []int, delim string) string {
	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
}

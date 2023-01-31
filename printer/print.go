package printer

import (
	"fmt"
	"os"

	"github.com/nanovms/ops/log"
)

func Debugf(format string, a ...interface{}) {
	fmt.Printf(log.ConsoleColors.Cyan()+format+log.ConsoleColors.Reset()+"\n", a...)
}

func Infof(format string, a ...interface{}) {
	fmt.Printf(log.ConsoleColors.Blue()+format+log.ConsoleColors.Reset()+"\n", a...)
}

func Warningf(format string, a ...interface{}) {
	fmt.Printf(log.ConsoleColors.Yellow()+format+log.ConsoleColors.Reset()+"\n", a...)
}

func Errorf(format string, a ...interface{}) {
	fmt.Printf(log.ConsoleColors.Red()+format+log.ConsoleColors.Reset()+"\n", a...)
}

func Fatalf(format string, a ...interface{}) {
	fmt.Printf(log.ConsoleColors.Red()+format+log.ConsoleColors.Reset()+"\n", a...)
	os.Exit(1)
}

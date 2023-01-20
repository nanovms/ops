package testutils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

const (
	basicProgram = `package main

import(
	"fmt"
)

func main(){
	fmt.Println("hello world")
}
`
)

// BuildBasicProgram generates binary from a hello world golang program in the current directory
func BuildBasicProgram() (binaryPath string) {
	program := []byte(basicProgram)
	randomString := String(5)
	binaryPath = "./basic" + randomString
	sourcePath := fmt.Sprintf("basic%s.go", randomString)

	err := os.WriteFile(sourcePath, program, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer os.Remove(sourcePath)

	cmd := exec.Command("go", "build", sourcePath)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GOOS=linux")
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return
}

//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package qemu

import (
	"fmt"
	"net"
	"os"
)

// turn me private and have the actual commands be exportable
func ExecuteQMP(commands []string, last string) {
	c, err := net.Dial("tcp", "localhost:"+last)
	if err != nil {
		fmt.Println(err)
	}
	defer c.Close()

	for i := 0; i < len(commands); i++ {
		_, err := c.Write([]byte(commands[i] + "\n"))
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("wrote %s\n", commands[i])
		received := make([]byte, 1024)
		_, err = c.Read(received)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

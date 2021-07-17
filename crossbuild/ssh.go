package crossbuild

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

// Creates new ssh client connected to given port using given credentials.
func newSSHClient(port int, username, password string) (*ssh.Client, error) {
	return ssh.Dial("tcp", fmt.Sprintf("localhost:%v", port), &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
}

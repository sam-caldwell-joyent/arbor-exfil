package cmd

import "golang.org/x/crypto/ssh"

// sshClientWrapper adapts *ssh.Client to sessionClient
type sshClientWrapper struct {
	c *ssh.Client
}

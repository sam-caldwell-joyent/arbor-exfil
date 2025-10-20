package cmd

import "golang.org/x/crypto/ssh"

// sshClientWrapper adapts *ssh.Client to the sessionClient interface. It
// provides a uniform way to create sessions regardless of whether the caller
// is using plain exec channels or the persistent shell machinery.
type sshClientWrapper struct {
	c *ssh.Client
}

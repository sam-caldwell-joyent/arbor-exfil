package cmd

import "golang.org/x/crypto/ssh"

// sshSessionWrapper adapts *ssh.Session to the internal session interface so
// callers can remain oblivious to the concrete SSH transport.
type sshSessionWrapper struct {
	s *ssh.Session
}

package cmd

import "golang.org/x/crypto/ssh"

// sshSessionWrapper adapts *ssh.Session to session
type sshSessionWrapper struct {
	s *ssh.Session
}

package cmd

import "fmt"

// NewSession opens a new exec channel on the underlying *ssh.Client and wraps
// it in a session-compatible adapter so the rest of the code can remain
// transport-agnostic.
func (w sshClientWrapper) NewSession() (session, error) {
	if w.c == nil {
		return nil, fmt.Errorf("nil ssh client")
	}
	s, err := w.c.NewSession()
	if err != nil {
		return nil, err
	}
	return sshSessionWrapper{s}, nil
}

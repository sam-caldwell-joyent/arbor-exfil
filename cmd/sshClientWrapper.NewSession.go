package cmd

import "fmt"

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

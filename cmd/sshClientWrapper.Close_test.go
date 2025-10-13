package cmd

import "testing"

type _closer interface{ Close() error }

// Presence test: ensure method exists on the wrapper type
func TestSSHSessionWrapper_Close_Present(t *testing.T) {
    var _ _closer = sshSessionWrapper{}
}


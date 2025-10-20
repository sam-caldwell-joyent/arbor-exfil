package cmd

import "testing"

type _closer interface{ Close() error }

// TestSSHSessionWrapper_Close_Present verifies at compile time that
// sshSessionWrapper implements Close, guarding against accidental removal.
// Assumes no runtime behavior is executed.
func TestSSHSessionWrapper_Close_Present(t *testing.T) {
    var _ _closer = sshSessionWrapper{}
}

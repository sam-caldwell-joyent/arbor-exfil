package cmd

import "testing"

type _combiner interface{ CombinedOutput(string) ([]byte, error) }

// Presence test: ensure method exists on the wrapper type
func TestSSHSessionWrapper_CombinedOutput_Present(t *testing.T) {
    var _ _combiner = sshSessionWrapper{}
}


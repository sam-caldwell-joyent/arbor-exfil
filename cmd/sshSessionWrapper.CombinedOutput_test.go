package cmd

import "testing"

type _combiner interface{ CombinedOutput(string) ([]byte, error) }

// TestSSHSessionWrapper_CombinedOutput_Present verifies at compile time that
// sshSessionWrapper implements CombinedOutput, guarding against method removal.
// Assumes no runtime behavior is executed.
func TestSSHSessionWrapper_CombinedOutput_Present(t *testing.T) {
    var _ _combiner = sshSessionWrapper{}
}

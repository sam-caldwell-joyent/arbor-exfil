package cmd

import "testing"

// TestSSHSessionWrapper_Type_Exists verifies that the sshSessionWrapper type is
// present and can be instantiated in tests. Assumes no runtime behavior is
// exercised here (type existence compile check only).
func TestSSHSessionWrapper_Type_Exists(t *testing.T) {
    _ = sshSessionWrapper{}
}

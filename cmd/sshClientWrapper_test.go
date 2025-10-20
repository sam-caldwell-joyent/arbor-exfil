package cmd

import "testing"

// TestSSHClientWrapper_Type_Exists verifies that the sshClientWrapper type is
// present and can be instantiated for tests. Assumes a compile-time check only.
func TestSSHClientWrapper_Type_Exists(t *testing.T) {
    _ = sshClientWrapper{}
}

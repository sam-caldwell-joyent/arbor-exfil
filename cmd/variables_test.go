package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

// TestVariables_DefaultsAndSwap verifies that important package-level
// variables are initialized (error messages and function pointers) and can be
// safely swapped for tests. Assumes default initialization occurred in init().
func TestVariables_DefaultsAndSwap(t *testing.T) {
    // errAdminUser message
    require.Contains(t, errAdminUser.Error(), "admin user")

    // dialSSHFunc and runRemoteCommandFunc are set
    require.NotNil(t, dialSSHFunc)
    require.NotNil(t, runRemoteCommandFunc)

    // Can be swapped
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun })
}

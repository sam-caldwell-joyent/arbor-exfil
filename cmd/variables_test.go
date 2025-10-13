package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

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


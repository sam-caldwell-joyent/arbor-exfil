package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestSSHSessionWrapper_CombinedOutput_PanicsOnNil(t *testing.T) {
    var w sshSessionWrapper // zero value has nil s
    require.Panics(t, func() { _, _ = w.CombinedOutput("echo") })
}


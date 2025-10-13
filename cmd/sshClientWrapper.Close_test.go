package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestSSHSessionWrapper_Close_PanicsOnNil(t *testing.T) {
    var w sshSessionWrapper // zero value has nil s
    require.Panics(t, func() { _ = w.Close() })
}


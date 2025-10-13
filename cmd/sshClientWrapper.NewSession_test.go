package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestSSHClientWrapper_NewSession_NilClientError(t *testing.T) {
    var w sshClientWrapper
    s, err := w.NewSession()
    require.Error(t, err)
    require.Nil(t, s)
}


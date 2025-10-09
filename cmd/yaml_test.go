package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestYAML_UnmarshalErrors(t *testing.T) {
    var mf manifest
    // Invalid type for commands should error
    err := yamlUnmarshalImpl([]byte("commands: 123"), &mf)
    require.Error(t, err)
}

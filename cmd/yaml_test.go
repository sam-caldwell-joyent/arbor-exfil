package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

// TestYAML_UnmarshalErrors verifies that yamlUnmarshalImpl returns an error
// when provided YAML does not match the expected structure. Assumes minimal
// invalid input.
func TestYAML_UnmarshalErrors(t *testing.T) {
    var mf manifest
    // Invalid type for commands should error
    err := yamlUnmarshalImpl([]byte("commands: 123"), &mf)
    require.Error(t, err)
}

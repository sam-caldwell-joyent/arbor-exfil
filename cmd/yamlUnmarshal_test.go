package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestYAML_Unmarshal_Success_Dedicated(t *testing.T) {
    var mf manifest
    data := []byte("name: N\ndescription: D\ncommands:\n  - command: x\n")
    err := yamlUnmarshalImpl(data, &mf)
    require.NoError(t, err)
    require.Equal(t, "N", mf.Name)
    require.Equal(t, 1, len(mf.Commands))
}

func TestYAML_Unmarshal_Errors_Dedicated(t *testing.T) {
    var mf manifest
    err := yamlUnmarshalImpl([]byte("commands: 123"), &mf)
    require.Error(t, err)
}

func TestCommandEntry_UnmarshalYAML_DecodeError(t *testing.T) {
    var ce commandEntry
    // A scalar where a mapping is expected should cause Decode error inside UnmarshalYAML
    err := yamlUnmarshalImpl([]byte("- 123"), &[]commandEntry{ce})
    require.Error(t, err)
}

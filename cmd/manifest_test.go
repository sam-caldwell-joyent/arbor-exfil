package cmd

import (
    "testing"
    "gopkg.in/yaml.v3"
    "github.com/stretchr/testify/require"
)

func TestManifest_YAML_Unmarshal_Structure(t *testing.T) {
    var m manifest
    data := []byte("name: N\ndescription: D\ncommands:\n  - command: x\n")
    require.NoError(t, yaml.Unmarshal(data, &m))
    require.Equal(t, "N", m.Name)
    require.Equal(t, "D", m.Description)
    require.Equal(t, 1, len(m.Commands))
    require.Equal(t, "x", m.Commands[0].Command)
}


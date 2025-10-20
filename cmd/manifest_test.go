package cmd

import (
    "testing"
    "gopkg.in/yaml.v3"
    "github.com/stretchr/testify/require"
)

// TestManifest_YAML_Unmarshal_Structure verifies that top-level manifest fields
// and a basic command decode correctly from YAML. Assumes minimal YAML.
func TestManifest_YAML_Unmarshal_Structure(t *testing.T) {
    var m manifest
    data := []byte("name: N\ndescription: D\ncommands:\n  - command: x\n")
    require.NoError(t, yaml.Unmarshal(data, &m))
    require.Equal(t, "N", m.Name)
    require.Equal(t, "D", m.Description)
    require.Equal(t, 1, len(m.Commands))
    require.Equal(t, "x", m.Commands[0].Command)
}

// TestManifest_YAML_SSHHost_Optional verifies that the optional ssh_host
// structure decodes when present, providing defaults for run/install.
func TestManifest_YAML_SSHHost_Optional(t *testing.T) {
    var m manifest
    data := []byte("name: N\ndescription: D\nssh_host:\n  ip: 10.0.0.1\n  user: bob\ncommands:\n  - command: x\n")
    require.NoError(t, yaml.Unmarshal(data, &m))
    require.Equal(t, "N", m.Name)
    require.Equal(t, "D", m.Description)
    require.Equal(t, "10.0.0.1", m.SSHHost.IP)
    require.Equal(t, "bob", m.SSHHost.User)
    require.Equal(t, 1, len(m.Commands))
}

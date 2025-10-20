package cmd

import (
    "testing"
    "gopkg.in/yaml.v3"
    "github.com/stretchr/testify/require"
)

// TestCommandEntry_YAML_Unmarshal_CommandAndArgs verifies that command and args
// fields are parsed from YAML and preserved. Assumes no shell field provided.
func TestCommandEntry_YAML_Unmarshal_CommandAndArgs(t *testing.T) {
    var c commandEntry
    data := []byte("command: show version\nargs: ['a', 'b c']\n")
    require.NoError(t, yaml.Unmarshal(data, &c))
    require.Equal(t, "show version", c.Command)
    require.Equal(t, []string{"a", "b c"}, c.Args)
}

// TestCommandEntry_YAML_Unmarshal_ShellAccepted verifies that the shell field
// is accepted and stored on commandEntry during YAML unmarshal. Assumes valid
// command provided.
func TestCommandEntry_YAML_Unmarshal_ShellAccepted(t *testing.T) {
    var c commandEntry
    data := []byte("command: whoami\nshell: /bin/sh\n")
    require.NoError(t, yaml.Unmarshal(data, &c))
    require.Equal(t, "whoami", c.Command)
    require.Equal(t, "/bin/sh", c.Shell)
}

package cmd

import (
    "testing"
    "gopkg.in/yaml.v3"
    "github.com/stretchr/testify/require"
)

func TestCommandEntry_YAML_Unmarshal_CommandAndArgs(t *testing.T) {
    var c commandEntry
    data := []byte("command: show version\nargs: ['a', 'b c']\n")
    require.NoError(t, yaml.Unmarshal(data, &c))
    require.Equal(t, "show version", c.Command)
    require.Equal(t, []string{"a", "b c"}, c.Args)
}

func TestCommandEntry_YAML_Unmarshal_ShellAccepted(t *testing.T) {
    var c commandEntry
    data := []byte("command: whoami\nshell: /bin/sh\n")
    require.NoError(t, yaml.Unmarshal(data, &c))
    require.Equal(t, "whoami", c.Command)
    require.Equal(t, "/bin/sh", c.Shell)
}

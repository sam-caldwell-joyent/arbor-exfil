package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestCommandEntry_Line_NoArgs(t *testing.T) {
    c := commandEntry{Command: "whoami"}
    require.Equal(t, "whoami", c.line())
}

func TestCommandEntry_Line_WithQuoting(t *testing.T) {
    c := commandEntry{Command: "echo", Args: []string{"hello world", "a'b", "/ok/path"}}
    require.Equal(t, "echo 'hello world' 'a'\''b' /ok/path", c.line())
}


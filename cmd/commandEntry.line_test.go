package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandEntry_Line_NoArgs(t *testing.T) {
	c := commandEntry{Command: "whoami"}
	require.Equal(t, "whoami", c.line())
}

// TestCommandEntry_Line_WithQuoting verifies that args are shell-quoted when
// rendered into a single command line, preserving spaces and embedded quotes.
// Assumes shellQuote behavior is consistent with tests in shellQuote_test.go.
func TestCommandEntry_Line_WithQuoting(t *testing.T) {
	c := commandEntry{Command: "echo", Args: []string{"hello world", "a'b", "/ok/path"}}
	expected := "echo 'hello world' " + `"'a'\''b'"`[1:len(`"'a'\''b'"`)-1] + " /ok/path"
	require.Equal(t, expected, c.line())
}

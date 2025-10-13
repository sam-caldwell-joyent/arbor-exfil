package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestShellQuote_Dedicated(t *testing.T) {
    require.Equal(t, "simple", shellQuote("simple"))
    require.Equal(t, "''", shellQuote(""))
    require.Equal(t, "'two words'", shellQuote("two words"))
    require.Equal(t, `"'a'\''b'"`[1:len(`"'a'\''b'"`)-1], shellQuote("a'b"))
    require.Equal(t, "/path/ok", shellQuote("/path/ok"))
    require.Equal(t, "abc+123", shellQuote("abc+123"))
}


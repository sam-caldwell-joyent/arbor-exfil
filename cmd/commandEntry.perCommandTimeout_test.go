package cmd

import (
    "testing"
    "time"
    "github.com/stretchr/testify/require"
)

func TestCommandEntry_PerCommandTimeout_Cases(t *testing.T) {
    var c commandEntry
    def := 10 * time.Second
    require.Equal(t, def, c.perCommandTimeout(def))
    c.Timeout = "bad"
    require.Equal(t, def, c.perCommandTimeout(def))
    c.Timeout = "1s"
    require.Equal(t, 1*time.Second, c.perCommandTimeout(def))
}


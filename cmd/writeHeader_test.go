package cmd

import (
    "bytes"
    "testing"
    "github.com/stretchr/testify/require"
)

func TestWriteHeader_Dedicated(t *testing.T) {
    var buf bytes.Buffer
    mf := &manifest{Name: "N", Description: "D", Commands: []commandEntry{{Command: "x"}}}
    writeHeader(&buf, mf)
    s := buf.String()
    require.Contains(t, s, "Name: N")
    require.Contains(t, s, "Description: D")
    require.Contains(t, s, "Command Count: 1")
}


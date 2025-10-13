package cmd

import (
    "bytes"
    "errors"
    "testing"
    "time"
    "github.com/stretchr/testify/require"
)

func TestWriteCommandSection_Dedicated(t *testing.T) {
    var buf bytes.Buffer
    err := writeCommandSection(&buf, commandEntry{Command: "x"}, []byte("hello\n"), 0, nil, 0)
    require.NoError(t, err)
    out := buf.String()
    require.Contains(t, out, "Command: x\n")
    require.Contains(t, out, "Exit Code: 0")
    require.Contains(t, out, "---8<---\nhello\n---8<---")

    buf.Reset()
    err = writeCommandSection(&buf, commandEntry{Command: "x"}, []byte("no-nl"), 2, errors.New("boom"), 3*time.Second)
    require.NoError(t, err)
    out = buf.String()
    require.Contains(t, out, "Timeout: 3s")
    require.Contains(t, out, "Exit Code: 2")
    require.Contains(t, out, "Error: boom")
}


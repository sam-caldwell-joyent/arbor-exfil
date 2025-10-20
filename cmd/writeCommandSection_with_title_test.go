package cmd

import (
    "bytes"
    "testing"
    "github.com/stretchr/testify/require"
    "gopkg.in/yaml.v3"
)

// TestYAMLCommand_WithTitle verifies that a command result with a title is
// serialized into the YAML report and appears under the correct host/run.
// Assumes in-memory buffer for writing.
func TestYAMLCommand_WithTitle(t *testing.T) {
    rep := newYAMLReport(&manifest{Name: "N", Description: "D"})
    rep.addResult("child-1", yamlCmdResult{Title: "My Task", Command: "echo 1", ExitCode: 0, Output: "ok\n"})
    var buf bytes.Buffer
    require.NoError(t, writeYAMLReport(&buf, rep))
    var got yamlReport
    require.NoError(t, yaml.Unmarshal(buf.Bytes(), &got))
    require.Equal(t, 1, len(got.Runs))
    require.Equal(t, "child-1", got.Runs[0].Host)
    require.Equal(t, 1, len(got.Runs[0].Results))
    require.Equal(t, "My Task", got.Runs[0].Results[0].Title)
    require.Equal(t, "echo 1", got.Runs[0].Results[0].Command)
}

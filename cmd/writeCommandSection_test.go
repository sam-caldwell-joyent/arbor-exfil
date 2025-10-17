package cmd

import (
    "bytes"
    "testing"
    "github.com/stretchr/testify/require"
    "gopkg.in/yaml.v3"
)

func TestYAMLCommandResult_Dedicated(t *testing.T) {
    rep := newYAMLReport(&manifest{Name: "N", Description: "D"})
    rep.addResult("", yamlCmdResult{Command: "x", ExitCode: 0, Output: "hello\n"})
    rep.addResult("", yamlCmdResult{Command: "x", ExitCode: 2, Timeout: "3s", Error: "boom", Output: "no-nl"})
    var buf bytes.Buffer
    require.NoError(t, writeYAMLReport(&buf, rep))
    var got yamlReport
    require.NoError(t, yaml.Unmarshal(buf.Bytes(), &got))
    require.Equal(t, 1, len(got.Runs))
    require.Equal(t, 2, len(got.Runs[0].Results))
    require.Equal(t, "x", got.Runs[0].Results[0].Command)
    require.Equal(t, 0, got.Runs[0].Results[0].ExitCode)
    require.Equal(t, "hello\n", got.Runs[0].Results[0].Output)
    require.Equal(t, "3s", got.Runs[0].Results[1].Timeout)
    require.Equal(t, "boom", got.Runs[0].Results[1].Error)
}

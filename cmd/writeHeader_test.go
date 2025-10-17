package cmd

import (
    "bytes"
    "testing"
    "github.com/stretchr/testify/require"
    "gopkg.in/yaml.v3"
)

func TestYAMLHeader_Dedicated(t *testing.T) {
    var buf bytes.Buffer
    mf := &manifest{Name: "N", Description: "D"}
    rep := newYAMLReport(mf)
    require.NoError(t, writeYAMLReport(&buf, rep))
    var got yamlReport
    require.NoError(t, yaml.Unmarshal(buf.Bytes(), &got))
    require.Equal(t, "N", got.Name)
    require.Equal(t, "D", got.Description)
}

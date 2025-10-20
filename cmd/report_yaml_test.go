package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
    "gopkg.in/yaml.v3"
)

// TestYAMLReport_AddRunMerge_SetDiscoveryVariants verifies discovery fields,
// run merging by host, and YAML round-trip correctness. Assumes in-memory data
// and no I/O.
func TestYAMLReport_AddRunMerge_SetDiscoveryVariants(t *testing.T) {
    mf := &manifest{Name: "N", Description: "D"}
    rep := newYAMLReport(mf)

    // includeContent=false: hosts_content stays empty
    rep.setDiscovery([]byte("127.0.0.1 localhost\n"), []string{"127.0.0.1"}, false)
    require.NotNil(t, rep.Discovery)
    require.Equal(t, "", rep.Discovery.HostsContent)
    require.Equal(t, []string{"127.0.0.1"}, rep.Discovery.DiscoveredHosts)

    // includeContent=true: hosts_content captured
    rep.setDiscovery([]byte("127.0.0.1 localhost\n10.0.0.2 node\n"), []string{"127.0.0.1", "10.0.0.2"}, true)
    require.Contains(t, rep.Discovery.HostsContent, "10.0.0.2")

    // addRun merges by host
    r1 := rep.addRun("10.0.0.2")
    r1.Results = append(r1.Results, yamlCmdResult{Command: "x", ExitCode: 0})
    r2 := rep.addRun("10.0.0.3")
    r2.Results = append(r2.Results, yamlCmdResult{Command: "y", ExitCode: 1})
    r1b := rep.addRun("10.0.0.2")
    require.Equal(t, r1, r1b)
    rep.addResult("10.0.0.2", yamlCmdResult{Command: "z", ExitCode: 0})

    // Marshal/unmarshal round trip
    b, err := yaml.Marshal(rep)
    require.NoError(t, err)
    var got yamlReport
    require.NoError(t, yaml.Unmarshal(b, &got))
    require.Equal(t, 2, len(got.Runs))
}

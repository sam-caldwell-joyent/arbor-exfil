package cmd

import (
    "os"
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/require"
    "golang.org/x/crypto/ssh"
    "gopkg.in/yaml.v3"
    "time"
)

// TestDiscovery_WritesSection_AndChildHeaders verifies that the run subcommand
// performs discovery, records the discovered hosts (excluding loopback), and
// then runs commands for each host using the specified shell. Assumes stubbed
// dial/run and a manifest with ssh_host defaults.
func TestDiscovery_WritesSection_AndChildHeaders(t *testing.T) {
    resetConfig()
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun })

    var issued []string
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        issued = append(issued, cmd)
        if len(issued) == 1 {
            return []byte("10.0.0.1 node1\n10.0.0.2 node2\n"), 0, nil
        }
        return []byte("ok\n"), 0, nil
    }
    dialSSHFunc = func(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
        return nil, nil
    }

    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
ssh_host:
  ip: 127.0.0.1
  user: tester
commands:
  - title: T
    command: whoami
    shell: /bin/comsh
`)
    outPath := filepath.Join(tmp, "out.txt")

    rootCmd.SetArgs([]string{
        "run",
        "--manifest", manifestPath,
        "--out", outPath,
        "--strict-host-key=false",
    })
    err := rootCmd.Execute()
    require.NoError(t, err)

    require.Equal(t, "cat /etc/hosts", issued[0])
    require.Contains(t, issued[1], "sudo -u admin --shell /bin/comsh -c 'whoami'")

    b, err := os.ReadFile(outPath)
    require.NoError(t, err)
    var rep yamlReport
    require.NoError(t, yaml.Unmarshal(b, &rep))
    require.ElementsMatch(t, []string{"10.0.0.1", "10.0.0.2"}, rep.Discovery.DiscoveredHosts)
    // With commands present, hosts_content should be captured
    require.NotEmpty(t, rep.Discovery.HostsContent)
    // Runs for each child host
    require.Equal(t, 2, len(rep.Runs))
    hosts := []string{rep.Runs[0].Host, rep.Runs[1].Host}
    require.ElementsMatch(t, []string{"10.0.0.1", "10.0.0.2"}, hosts)
}

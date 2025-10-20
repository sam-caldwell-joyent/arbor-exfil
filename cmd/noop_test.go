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

// TestNoop_WritesPlannedCommandsToDebugOut verifies that when --noop is set,
// the CLI writes the expanded per-host command lines to debug.out and still
// emits a YAML report containing discovery results. Assumes stubbed dial/run
// functions and a manifest specifying shells for commands.
func TestNoop_WritesPlannedCommandsToDebugOut(t *testing.T) {
    resetConfig()
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun })

    calls := 0
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        calls++
        if calls == 1 {
            return []byte("127.0.0.1 localhost\n10.0.0.1 node1\n10.0.0.2 node2\n10.0.0.1 node1dup\n"), 0, nil
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
  - title: A
    command: cmd1
    shell: /bin/sh
  - title: B
    command: cmd2
    shell: /bin/sh
`)
    outPath := filepath.Join(tmp, "out.txt")

    rootCmd.SetArgs([]string{
        "run",
        "--manifest", manifestPath,
        "--out", outPath,
        "--strict-host-key=false",
        "--noop",
    })
    err := rootCmd.Execute()
    require.NoError(t, err)

    b, err := os.ReadFile("debug.out")
    require.NoError(t, err)
    s := string(b)
    require.Contains(t, s, "sudo -u admin --shell /bin/sh -c 'cmd1'")
    require.Contains(t, s, "sudo -u admin --shell /bin/sh -c 'cmd2'")

    // Also verify YAML report was written with discovery and no runs
    outYaml, err := os.ReadFile(outPath)
    require.NoError(t, err)
    var rep yamlReport
    require.NoError(t, yaml.Unmarshal(outYaml, &rep))
    require.Nil(t, rep.Runs)
    require.NotNil(t, rep.Discovery)
    require.GreaterOrEqual(t, len(rep.Discovery.DiscoveredHosts), 1)
}

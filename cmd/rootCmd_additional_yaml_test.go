package cmd

import (
    "os"
    "testing"
    "github.com/stretchr/testify/require"
    "golang.org/x/crypto/ssh"
    "gopkg.in/yaml.v3"
    "time"
)

// TestRootExecute_EmptyDiscovery_RunsOnceOnLeader verifies that when discovery
// returns no child hosts, commands are executed once on the leader and the
// YAML report records an empty host for that run. Assumes stubbed dial/run.
func TestRootExecute_EmptyDiscovery_RunsOnceOnLeader(t *testing.T) {
    resetConfig()
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun })

    dialSSHFunc = func(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
        return nil, nil
    }
    calls := 0
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        calls++
        if calls == 1 {
            return []byte(""), 0, nil // discovery returns no hosts
        }
        return []byte("ok\n"), 0, nil
    }

    tmp := t.TempDir()
    mf := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: whoami
    shell: /bin/sh
`)
    out := tmp + "/out.yaml"
    rootCmd.SetArgs([]string{"run","--target","127.0.0.1:22","--user","u","--manifest", mf, "--out", out, "--strict-host-key=false"})
    require.NoError(t, rootCmd.Execute())
    b, err := os.ReadFile(out)
    require.NoError(t, err)
    var rep yamlReport
    require.NoError(t, yaml.Unmarshal(b, &rep))
    require.Equal(t, 1, len(rep.Runs))
    require.Equal(t, "", rep.Runs[0].Host)
    require.Equal(t, 1, len(rep.Runs[0].Results))
}

// TestRootExecute_PerCommandTimeout_IncludedInYAML verifies that a per-command
// timeout specified in the manifest is reflected in the YAML report and that a
// deadline-exceeded error is recorded when simulated. Assumes stubbed run.
func TestRootExecute_PerCommandTimeout_IncludedInYAML(t *testing.T) {
    resetConfig()
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun })

    dialSSHFunc = func(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
        return nil, nil
    }
    calls := 0
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        calls++
        if calls == 1 {
            return []byte("127.0.0.1 localhost\n"), 0, nil // discovery
        }
        return nil, -1, contextDeadlineErr()
    }

    tmp := t.TempDir()
    mf := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: slow
    shell: /bin/sh
    timeout: 1s
`)
    out := tmp + "/out.yaml"
    rootCmd.SetArgs([]string{"run","--target","127.0.0.1:22","--user","u","--manifest", mf, "--out", out, "--strict-host-key=false"})
    require.NoError(t, rootCmd.Execute())
    b, err := os.ReadFile(out)
    require.NoError(t, err)
    var rep yamlReport
    require.NoError(t, yaml.Unmarshal(b, &rep))
    require.GreaterOrEqual(t, len(rep.Runs), 1)
    found := false
    for _, r := range rep.Runs[0].Results {
        if r.Command == "slow" {
            require.Equal(t, "1s", r.Timeout)
            require.Equal(t, -1, r.ExitCode)
            require.Contains(t, r.Error, "deadline exceeded")
            found = true
        }
    }
    require.True(t, found)
}

// contextDeadlineErr returns a context deadline exceeded error without importing context here
type deadErr struct{}
func (deadErr) Error() string { return "context deadline exceeded" }
func contextDeadlineErr() error { return deadErr{} }

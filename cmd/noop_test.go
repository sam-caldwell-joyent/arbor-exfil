package cmd

import (
    "os"
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/require"
    "golang.org/x/crypto/ssh"
    "time"
)

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
}

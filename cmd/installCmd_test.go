package cmd

import (
    "testing"
    "time"
    "strings"

    "github.com/stretchr/testify/require"
    "golang.org/x/crypto/ssh"
)

// TestInstall_FiltersLoopbacks_AndInstallsKey verifies that the install
// subcommand performs discovery on the leader, filters loopback addresses, and
// issues install commands (via sudo) to each discovered host. Assumes stubbed
// dial/run functions and a temporary manifest and public key file.
func TestInstall_FiltersLoopbacks_AndInstallsKey(t *testing.T) {
    resetConfig()
    tmp := t.TempDir()
    // Write a fake pubkey
    pub := writeTemp(t, tmp, "id.pub", "ssh-ed25519 AAAAB3NzaC1lZDI1NTE5AAAAITestKey arbor-exfil")
    // Write a manifest with ssh_host defaults
    mf := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
ssh_host:
  ip: 127.0.0.1
  user: admin
`)

    // Stub dial and run to avoid network
    var dials []string
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun })

    // First dial is leader at 127.0.0.1:22, then hosts 10.0.0.1:22 and 10.0.0.2:22
    dialSSHFunc = func(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
        dials = append(dials, target+","+user)
        return nil, nil
    }

    calls := 0
    var issued []string
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        calls++
        issued = append(issued, cmd)
        if calls == 1 {
            // discovery on leader; include loopbacks which must be filtered
            return []byte("127.0.0.1 localhost\n::1 ip6-localhost\n10.0.0.1 node1\n10.0.0.2 node2\n"), 0, nil
        }
        // For install commands on hosts, return success
        return []byte("ok\n"), 0, nil
    }

    rootCmd.SetArgs([]string{"install", "--manifest", mf, "--install-pubkey", pub, "--strict-host-key=false"})
    err := rootCmd.Execute()
    require.NoError(t, err)

    // Expect one dial to leader and two dials to discovered hosts (loopbacks filtered)
    require.GreaterOrEqual(t, len(dials), 3)
    require.Contains(t, dials[0], "127.0.0.1:22")
    // Ensure issued install command contains sudo and arbor-exfil
    foundInstall := false
    for _, c := range issued[1:] { // skip discovery
        if strings.Contains(c, "sudo -n /bin/sh -c") && strings.Contains(c, "arbor-exfil") {
            foundInstall = true
        }
    }
    require.True(t, foundInstall)
}

// TestInstall_RequiresManifest_AndPubkey verifies that the install subcommand
// enforces required flags: --manifest and --install-pubkey must be provided.
// Assumes default rootCmd config and no network dependencies.
func TestInstall_RequiresManifest_AndPubkey(t *testing.T) {
    resetConfig()
    rootCmd.SetArgs([]string{"install"})
    err := rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "--manifest is required")

    resetConfig()
    tmp := t.TempDir()
    mf := writeTemp(t, tmp, "m.yaml", "name: N\ndescription: D\nssh_host:\n  ip: 1.2.3.4\n  user: root\n")
    rootCmd.SetArgs([]string{"install", "--manifest", mf})
    err = rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "--install-pubkey is required")
}

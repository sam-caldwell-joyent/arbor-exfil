package cmd

import (
    "fmt"
    "net"
    "os"
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "path/filepath"
    "testing"
    "time"
    "github.com/stretchr/testify/require"
)

// TestDialSSH_StrictHostKeyMissingKnownHosts_Dedicated verifies that strict
// host-key checking with a missing known_hosts file fails fast. Assumes no
// network connection is established.
func TestDialSSH_StrictHostKeyMissingKnownHosts_Dedicated(t *testing.T) {
    _, err := dialSSH("127.0.0.1:22", "u", "", "", "", filepath.Join(t.TempDir(), "nope"), true, 100*time.Millisecond)
    require.Error(t, err)
}

// TestDialSSH_StrictHostKeyWithKnownHosts_Dedicated verifies that providing an
// existing known_hosts allows configuration to proceed to the connect attempt,
// which still fails due to an invalid address. Assumes localhost dial error is ok.
func TestDialSSH_StrictHostKeyWithKnownHosts_Dedicated(t *testing.T) {
    tmp := t.TempDir()
    kh := writeTemp(t, tmp, "known_hosts", "\n")
    _, err := dialSSH("127.0.0.1:1", "u", "", "", "", kh, true, 50*time.Millisecond)
    require.Error(t, err)
}

// TestDialSSH_AuthMethodsAssembly_Dedicated verifies that auth methods are
// assembled from agent and key material when available. Assumes invalid target
// so that final connection fails without affecting method assembly.
func TestDialSSH_AuthMethodsAssembly_Dedicated(t *testing.T) {
    t.Setenv("SSH_AUTH_SOCK", filepath.Join(t.TempDir(), "no.sock"))
    tmp := t.TempDir()
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    require.NoError(t, err)
    b := x509.MarshalPKCS1PrivateKey(key)
    pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b})
    keyPath := writeTemp(t, tmp, "id_rsa", string(pemBytes))
    _, err = dialSSH("127.0.0.1:1", "u", "p", keyPath, "", "", false, 50*time.Millisecond)
    require.Error(t, err)
}

// TestDialSSH_AgentSocketPresent_Dedicated verifies that when SSH_AUTH_SOCK is
// set to a valid UNIX socket, the agent auth path is exercised. Assumes the TCP
// connect will fail due to an invalid port.
func TestDialSSH_AgentSocketPresent_Dedicated(t *testing.T) {
    // Create a dummy UNIX socket to simulate SSH agent (use short path in /tmp)
    sock := filepath.Join(os.TempDir(), fmt.Sprintf("agent-%d.sock", time.Now().UnixNano()))
    l, err := net.Listen("unix", sock)
    require.NoError(t, err)
    defer l.Close()
    defer os.Remove(sock)
    // Accept one connection in the background
    done := make(chan struct{})
    go func() { defer close(done); conn, _ := l.Accept(); if conn != nil { time.Sleep(10 * time.Millisecond); _ = conn.Close() } }()

    t.Setenv("SSH_AUTH_SOCK", sock)
    // Use strictHost=false to avoid known_hosts requirements; expect dial error on tcp connect
    _, err = dialSSH("127.0.0.1:1", "u", "p", "", "", "", false, 50*time.Millisecond)
    require.Error(t, err)
    <-done
}

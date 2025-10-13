package cmd

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "path/filepath"
    "testing"
    "time"
    "github.com/stretchr/testify/require"
)

func TestDialSSH_StrictHostKeyMissingKnownHosts_Dedicated(t *testing.T) {
    _, err := dialSSH("127.0.0.1:22", "u", "", "", "", filepath.Join(t.TempDir(), "nope"), true, 100*time.Millisecond)
    require.Error(t, err)
}

func TestDialSSH_StrictHostKeyWithKnownHosts_Dedicated(t *testing.T) {
    tmp := t.TempDir()
    kh := writeTemp(t, tmp, "known_hosts", "\n")
    _, err := dialSSH("127.0.0.1:1", "u", "", "", "", kh, true, 50*time.Millisecond)
    require.Error(t, err)
}

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


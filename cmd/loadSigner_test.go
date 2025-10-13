package cmd

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/require"
)

func TestLoadSigner_FileNotFound_Dedicated(t *testing.T) {
    _, err := loadSigner(filepath.Join(t.TempDir(), "missing_key"), "")
    require.Error(t, err)
}

func TestLoadSigner_RSAKey_Success_Dedicated(t *testing.T) {
    tmp := t.TempDir()
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    require.NoError(t, err)
    b := x509.MarshalPKCS1PrivateKey(key)
    pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b})
    p := writeTemp(t, tmp, "id_rsa", string(pemBytes))
    s, err := loadSigner(p, "")
    require.NoError(t, err)
    require.NotNil(t, s.PublicKey())
}


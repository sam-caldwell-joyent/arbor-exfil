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

func TestLoadSigner_UnencryptedKey_WithPassphrase_Fails(t *testing.T) {
    tmp := t.TempDir()
    // Generate a small RSA key for testing (unencrypted)
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    require.NoError(t, err)
    b := x509.MarshalPKCS1PrivateKey(key)
    pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b})
    p := writeTemp(t, tmp, "id_rsa", string(pemBytes))
    // Provide a passphrase; parsing should fail for an unencrypted key
    _, err = loadSigner(p, "pass")
    require.Error(t, err)
}

func TestLoadSigner_EncryptedKey_MissingPassphrase_Error(t *testing.T) {
    tmp := t.TempDir()
    // Generate RSA key and encrypt PEM block with passphrase
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    require.NoError(t, err)
    der := x509.MarshalPKCS1PrivateKey(key)
    block, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", der, []byte("pp"), x509.PEMCipherAES256)
    require.NoError(t, err)
    pemBytes := pem.EncodeToMemory(block)
    p := writeTemp(t, tmp, "id_rsa_enc", string(pemBytes))
    // Missing passphrase should yield friendly error
    _, err = loadSigner(p, "")
    require.Error(t, err)
    require.Contains(t, err.Error(), "private key is encrypted")
}

func TestLoadSigner_EncryptedKey_WithPassphrase_Success(t *testing.T) {
    tmp := t.TempDir()
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    require.NoError(t, err)
    der := x509.MarshalPKCS1PrivateKey(key)
    block, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", der, []byte("pp"), x509.PEMCipherAES256)
    require.NoError(t, err)
    pemBytes := pem.EncodeToMemory(block)
    p := writeTemp(t, tmp, "id_rsa_enc", string(pemBytes))
    s, err := loadSigner(p, "pp")
    require.NoError(t, err)
    require.NotNil(t, s)
}

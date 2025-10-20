package cmd

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// loadSigner loads an SSH private key from path, returning a signer suitable
// for public key authentication. If a passphrase is provided, it attempts to
// decrypt the key; otherwise it tries an unencrypted parse and reports a clear
// error when an encrypted key requires a passphrase.
func loadSigner(path, passphrase string) (ssh.Signer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if passphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase(b, []byte(passphrase))
	}
	// Try without passphrase first
	s, err := ssh.ParsePrivateKey(b)
	if err == nil {
		return s, nil
	}
	// If passphrase missing, try empty
	var passphraseMissingError *ssh.PassphraseMissingError
	if errors.As(err, &passphraseMissingError) {
		return nil, fmt.Errorf("private key is encrypted; provide --passphrase or ARBOR_EXFIL_PASSPHRASE")
	}
	return nil, err
}

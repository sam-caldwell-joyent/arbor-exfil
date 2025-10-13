package cmd

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// dialSSH establishes an SSH client connection with options
func dialSSH(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
	var auths []ssh.AuthMethod

	if keyPath != "" {
		signer, err := loadSigner(keyPath, passphrase)
		if err != nil {
			return nil, fmt.Errorf("load key: %w", err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}

	if password != "" {
		auths = append(auths, ssh.Password(password))
	}

	// Try SSH agent if available
	if a := os.Getenv("SSH_AUTH_SOCK"); a != "" {
		if conn, err := net.Dial("unix", a); err == nil {
			ag := agent.NewClient(conn)
			auths = append(auths, ssh.PublicKeysCallback(ag.Signers))
		}
	}

	var hostKeyCB ssh.HostKeyCallback
	if strictHost {
		// Try known_hosts file if present; else fail closed
		if _, err := os.Stat(knownHostsPath); err == nil {
			cb, err := knownhosts.New(knownHostsPath)
			if err != nil {
				return nil, fmt.Errorf("known_hosts: %w", err)
			}
			hostKeyCB = cb
		} else {
			return nil, fmt.Errorf("known_hosts file not found at %s and strict-host-key is enabled", knownHostsPath)
		}
	} else {
		hostKeyCB = ssh.InsecureIgnoreHostKey()
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: hostKeyCB,
		Timeout:         dialTimeout,
	}

	// Use explicit net.Dialer for connection timeout
	d := net.Dialer{Timeout: dialTimeout}
	conn, err := d.Dial("tcp", target)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, target, cfg)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

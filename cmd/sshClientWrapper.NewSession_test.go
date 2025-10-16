package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
    "crypto/rand"
    "crypto/rsa"
    "net"
    "time"
    "golang.org/x/crypto/ssh"
)

func TestSSHClientWrapper_NewSession_NilClientError(t *testing.T) {
    var w sshClientWrapper
    s, err := w.NewSession()
    require.Error(t, err)
    require.Nil(t, s)
}

func TestSSHClientWrapper_NewSession_ServerRejects(t *testing.T) {
    // Start minimal SSH server that rejects session channel opens
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    require.NoError(t, err)
    defer ln.Close()
    done := make(chan struct{})
    go func() {
        defer close(done)
        key, _ := rsa.GenerateKey(rand.Reader, 1024)
        signer, _ := ssh.NewSignerFromKey(key)
        cfg := &ssh.ServerConfig{NoClientAuth: true}
        cfg.AddHostKey(signer)
        conn, err := ln.Accept()
        if err != nil { return }
        sc, chans, reqs, err := ssh.NewServerConn(conn, cfg)
        if err != nil { return }
        _ = sc
        go ssh.DiscardRequests(reqs)
        for ch := range chans {
            // Reject the first session open, then close the server connection
            // to ensure the client returns promptly instead of hanging.
            _ = ch.Reject(ssh.ResourceShortage, "no sessions")
            _ = sc.Close()
            break
        }
    }()
    cfg := &ssh.ClientConfig{User: "u", HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 2 * time.Second}
    client, err := ssh.Dial("tcp", ln.Addr().String(), cfg)
    require.NoError(t, err)
    defer client.Close()
    w := sshClientWrapper{c: client}
    s, err := w.NewSession()
    require.Error(t, err)
    require.Nil(t, s)
    <-done
}

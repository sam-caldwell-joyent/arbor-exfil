package cmd

import (
    "crypto/rand"
    "crypto/rsa"
    "net"
    "testing"
    "time"
    "golang.org/x/crypto/ssh"
)

func startNoopSessionServer(t *testing.T) (addr string, stop func()) {
    t.Helper()
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatalf("listen: %v", err) }
    done := make(chan struct{})
    go func() {
        defer close(done)
        priv, _ := rsa.GenerateKey(rand.Reader, 1024)
        signer, _ := ssh.NewSignerFromKey(priv)
        cfg := &ssh.ServerConfig{NoClientAuth: true}
        cfg.AddHostKey(signer)
        for {
            conn, err := ln.Accept()
            if err != nil { return }
            sc, chans, reqs, err := ssh.NewServerConn(conn, cfg)
            if err != nil { return }
            _ = sc
            go ssh.DiscardRequests(reqs)
            go func(chs <-chan ssh.NewChannel) {
                for ch := range chs {
                    if ch.ChannelType() != "session" { _ = ch.Reject(ssh.UnknownChannelType, ""); continue }
                    c, reqs, _ := ch.Accept()
                    go func(ch ssh.Channel, in <-chan *ssh.Request) {
                        defer ch.Close()
                        for req := range in {
                            // Accept pty and exec without doing anything
                            switch req.Type {
                            case "pty-req", "exec":
                                req.Reply(true, nil)
                            default:
                                req.Reply(false, nil)
                            }
                        }
                    }(c, reqs)
                }
            }(chans)
        }
    }()
    return ln.Addr().String(), func() { _ = ln.Close(); <-done }
}

// TestSSHClientWrapper_NewSession_Success_Integ verifies that NewSession on the
// wrapper returns a usable session against a minimal in-process SSH server
// that accepts PTY and exec. Assumes insecure host key callback for tests.
func TestSSHClientWrapper_NewSession_Success_Integ(t *testing.T) {
    addr, stop := startNoopSessionServer(t)
    defer stop()
    cfg := &ssh.ClientConfig{User: "u", HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 3 * time.Second}
    client, err := ssh.Dial("tcp", addr, cfg)
    if err != nil { t.Fatalf("dial: %v", err) }
    defer client.Close()
    w := sshClientWrapper{c: client}
    s, err := w.NewSession()
    if err != nil { t.Fatalf("NewSession error: %v", err) }
    if s == nil { t.Fatalf("expected non-nil session") }
}

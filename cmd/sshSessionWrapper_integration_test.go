package cmd

import (
    "crypto/rand"
    "crypto/rsa"
    "encoding/binary"
    "net"
    "strings"
    "testing"
    "time"
    "golang.org/x/crypto/ssh"
)

// startExecEchoServer starts a tiny SSH server that accepts exec requests and echoes arguments
func startExecEchoServer(t *testing.T) (addr string, stop func()) {
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
                            switch req.Type {
                            case "exec":
                                // parse command
                                var cmd string
                                if len(req.Payload) >= 4 {
                                    n := binary.BigEndian.Uint32(req.Payload)
                                    if int(n)+4 <= len(req.Payload) { cmd = string(req.Payload[4:4+n]) }
                                }
                                req.Reply(true, nil)
                                // very small echo handler
                                arg := strings.TrimSpace(strings.TrimPrefix(cmd, "echo "))
                                arg = strings.Trim(arg, "'\"")
                                ch.Write([]byte(arg+"\n"))
                                // send exit-status 0 then close
                                _, _ = ch.SendRequest("exit-status", false, []byte{0,0,0,0})
                                return
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

// TestSSHSessionWrapper_CombinedOutput_Integration verifies that the wrapper
// around *ssh.Session propagates CombinedOutput behavior against a tiny
// in-process SSH server that echoes its input. Assumes local TCP listen and
// insecure host key callbacks for test isolation.
func TestSSHSessionWrapper_CombinedOutput_Integration(t *testing.T) {
    addr, stop := startExecEchoServer(t)
    defer stop()
    cfg := &ssh.ClientConfig{User: "u", HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 3 * time.Second}
    client, err := ssh.Dial("tcp", addr, cfg)
    if err != nil { t.Fatalf("dial: %v", err) }
    defer client.Close()
    s, err := client.NewSession()
    if err != nil { t.Fatalf("session: %v", err) }
    w := sshSessionWrapper{s}
    out, err := w.CombinedOutput("echo hello")
    if err != nil { t.Fatalf("CombinedOutput: %v", err) }
    if string(out) != "hello\n" { t.Fatalf("unexpected out: %q", string(out)) }
    _ = w.Close()
}

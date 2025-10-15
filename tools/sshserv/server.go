package sshserv

import (
    "bufio"
    "crypto/rand"
    "crypto/rsa"
    "encoding/binary"
    "net"
    "strings"
    "time"

    "golang.org/x/crypto/ssh"
)

// Start launches a test SSH server listening on listenAddr (e.g., 127.0.0.1:20222).
// It accepts any user with no authentication and, for any exec session, emulates
// a POSIX shell that reads lines from stdin. For each line it writes:
//   ok\n
// and then echos a unique end marker with exit code 0 to delimit command output.
// The marker is extracted from a trailing "echo <marker> $?" appended by the client.
// Returns a stop function that closes the listener and waits for shutdown.
func Start(listenAddr string) (func(), error) {
    ln, err := net.Listen("tcp", listenAddr)
    if err != nil { return nil, err }

    stopCh := make(chan struct{})
    done := make(chan struct{})

    go func() {
        defer close(done)
        priv, _ := rsa.GenerateKey(rand.Reader, 2048)
        signer, _ := ssh.NewSignerFromKey(priv)
        cfg := &ssh.ServerConfig{NoClientAuth: true}
        cfg.AddHostKey(signer)

        for {
            _ = ln.(*net.TCPListener).SetDeadline(time.Now().Add(500 * time.Millisecond))
            conn, err := ln.Accept()
            select {
            case <-stopCh:
                if conn != nil { _ = conn.Close() }
                return
            default:
            }
            if err != nil {
                if ne, ok := err.(net.Error); ok && ne.Timeout() { continue }
                continue
            }
            go handleConn(conn, cfg)
        }
    }()

    stop := func() {
        close(stopCh)
        _ = ln.Close()
        <-done
    }
    return stop, nil
}

func handleConn(raw net.Conn, cfg *ssh.ServerConfig) {
    sc, chans, reqs, err := ssh.NewServerConn(raw, cfg)
    if err != nil { _ = raw.Close(); return }
    _ = sc
    go ssh.DiscardRequests(reqs)
    for ch := range chans {
        if ch.ChannelType() != "session" { _ = ch.Reject(ssh.UnknownChannelType, ""); continue }
        c, reqs, err := ch.Accept()
        if err != nil { continue }
        go handleSession(c, reqs)
    }
}

func handleSession(ch ssh.Channel, in <-chan *ssh.Request) {
    defer ch.Close()
    for req := range in {
        switch req.Type {
        case "pty-req":
            req.Reply(true, nil)
        case "shell":
            req.Reply(true, nil)
        case "exec":
            if len(req.Payload) >= 4 {
                n := binary.BigEndian.Uint32(req.Payload)
                _ = n
            }
            req.Reply(true, nil)
            emulateShell(ch)
            return
        default:
            req.Reply(false, nil)
        }
    }
}

func emulateShell(ch ssh.Channel) {
    br := bufio.NewReader(ch)
    for {
        line, err := br.ReadString('\n')
        if err != nil { return }
        s := strings.TrimSpace(line)
        if s == "" { continue }
        if s == "exit" { return }
        i := strings.LastIndex(s, "echo ")
        j := strings.LastIndex(s, " $?")
        marker := ""
        if i >= 0 && j > i+5 {
            marker = strings.TrimSpace(strings.Trim(s[i+5:j], "'\""))
        }
        _, _ = ch.Write([]byte("ok\n"))
        if marker != "" {
            _, _ = ch.Write([]byte(marker+" 0\n"))
        }
    }
}


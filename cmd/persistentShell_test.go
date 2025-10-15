package cmd

import (
    "bufio"
    "bytes"
    "crypto/rand"
    "crypto/rsa"
    "encoding/binary"
    "io"
    "net"
    "strings"
    "testing"
    "time"
    "golang.org/x/crypto/ssh"
)

// fakeShellWriter intercepts commands written to stdin and simulates the
// remote PTY by writing output and a marker line to pw.
type fakeShellWriter struct{
    pw *io.PipeWriter
    // customize payload and exit code per call
    payload string
    exit    int
}

func (w *fakeShellWriter) Write(p []byte) (int, error) {
    s := string(p)
    // Perform pipe writes asynchronously to avoid deadlock while caller
    // is still in Write and hasn't started reading.
    go func(cmd string) {
        // Extract marker between last "echo " and " $?"
        i := strings.LastIndex(cmd, "echo ")
        j := strings.LastIndex(cmd, " $?")
        marker := "__MARKER_MISSING__"
        if i >= 0 && j > i+len("echo ") {
            marker = strings.TrimSpace(cmd[i+len("echo ") : j])
            // Dequote if wrapped in single quotes
            marker = strings.Trim(marker, "'")
        }
        if w.payload != "" { _, _ = w.pw.Write([]byte(w.payload)) }
        // Emit marker and exit code
        _, _ = w.pw.Write([]byte(marker))
        _, _ = w.pw.Write([]byte{' '})
        _, _ = w.pw.Write([]byte(strings.TrimSpace(strconvItoa(w.exit))))
        _, _ = w.pw.Write([]byte{'\n'})
    }(s)
    return len(p), nil
}

func (w *fakeShellWriter) Close() error { _ = w.pw.Close(); return nil }

// small, dependency-free itoa for tests
func strconvItoa(n int) string {
    if n == 0 { return "0" }
    neg := false
    if n < 0 { neg = true; n = -n }
    var b [20]byte
    i := len(b)
    for n > 0 { i--; b[i] = byte('0' + (n%10)); n/=10 }
    if neg { i--; b[i] = '-' }
    return string(b[i:])
}

func TestPersistentShell_RunOne_SuccessExit0(t *testing.T) {
    pr, pw := io.Pipe()
    fw := &fakeShellWriter{pw: pw, payload: "hello\nworld\n", exit: 0}

    ps := &persistentShell{
        // sess is nil; we don't call Close() in this test
        stdin:  fw,
        pr:     pr,
        pw:     pw,
        reader: bufio.NewReader(pr),
        nonce:  "abc",
    }

    out, code, err := ps.runOne("echo ok")
    if err != nil { t.Fatalf("runOne error: %v", err) }
    if code != 0 { t.Fatalf("exit code = %d, want 0", code) }
    if string(out) != "hello\nworld\n" {
        t.Fatalf("output mismatch: %q", string(out))
    }
}

func TestPersistentShell_RunOne_NonZero_LongAccum(t *testing.T) {
    // Create payload > 8192 to exercise accum flush path
    var buf bytes.Buffer
    for i := 0; i < 9000; i++ { buf.WriteByte('A') }
    buf.WriteByte('\n')

    pr, pw := io.Pipe()
    fw := &fakeShellWriter{pw: pw, payload: buf.String(), exit: 2}

    ps := &persistentShell{
        stdin:  fw,
        pr:     pr,
        pw:     pw,
        reader: bufio.NewReader(pr),
        nonce:  "xyz",
    }

    out, code, err := ps.runOne("whoami")
    if err != nil { t.Fatalf("runOne error: %v", err) }
    if code != 2 { t.Fatalf("exit code = %d, want 2", code) }
    if len(out) < 9000 { t.Fatalf("expected long output, got %d bytes", len(out)) }
}

func TestParseInt(t *testing.T) {
    n, err := parseInt("0")
    if err != nil || n != 0 { t.Fatalf("parseInt(0) => %d,%v", n, err) }
    n, err = parseInt("123")
    if err != nil || n != 123 { t.Fatalf("parseInt(123) => %d,%v", n, err) }
    if _, err = parseInt("12x"); err == nil { t.Fatalf("expected error for invalid int") }
}

func TestPersistentSessionClient_NewSession_AndVirtualSession(t *testing.T) {
    pr, pw := io.Pipe()
    fw := &fakeShellWriter{pw: pw, payload: "out\n", exit: 7}
    ps := &persistentShell{
        stdin:  fw,
        pr:     pr,
        pw:     pw,
        reader: bufio.NewReader(pr),
        nonce:  "n",
    }
    client := persistentSessionClient{ps: ps}
    s, err := client.NewSession()
    if err != nil { t.Fatalf("NewSession error: %v", err) }
    b, err := s.CombinedOutput("cmd")
    if err != nil { t.Fatalf("CombinedOutput error: %v", err) }
    if string(b) != "out\n" { t.Fatalf("unexpected output: %q", string(b)) }
    // last exit code should be set on the concrete type
    if ec, ok := s.(interface{ LastExitCode() int }); ok {
        if ec.LastExitCode() != 7 { t.Fatalf("want exit 7, got %d", ec.LastExitCode()) }
    } else {
        t.Fatalf("session does not expose LastExitCode")
    }
    if err := s.Close(); err != nil { t.Fatalf("Close error: %v", err) }
}

func TestMakeNonce(t *testing.T) {
    n := makeNonce()
    if len(n) != 12 { t.Fatalf("nonce length = %d, want 12", len(n)) }
    for _, r := range n {
        if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') {
            t.Fatalf("nonce contains invalid char: %q", r)
        }
    }
}

// Integration test: exercise newPersistentShell using a minimal in-process SSH server
// that accepts PTY and exec, and simulates a shell by emitting the marker with
// an exit code after each stdin line.
func TestNewPersistentShell_Integration(t *testing.T) {
    // Start SSH server
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatalf("listen: %v", err) }
    defer ln.Close()

    serverDone := make(chan struct{})
    go func() {
        defer close(serverDone)
        priv, _ := rsa.GenerateKey(rand.Reader, 1024)
        signer, _ := ssh.NewSignerFromKey(priv)
        cfg := &ssh.ServerConfig{NoClientAuth: true}
        cfg.AddHostKey(signer)
        conn, err := ln.Accept()
        if err != nil { return }
        sc, chans, reqs, err := ssh.NewServerConn(conn, cfg)
        if err != nil { return }
        _ = sc
        go ssh.DiscardRequests(reqs)
        for ch := range chans {
            if ch.ChannelType() != "session" { _ = ch.Reject(ssh.UnknownChannelType, ""); continue }
            c, reqs, _ := ch.Accept()
            go func(ch ssh.Channel, in <-chan *ssh.Request) {
                defer ch.Close()
                for req := range in {
                    switch req.Type {
                    case "pty-req":
                        req.Reply(true, nil)
                    case "exec":
                        // parse command string (length-prefixed)
                        if len(req.Payload) >= 4 {
                            n := binary.BigEndian.Uint32(req.Payload)
                            _ = n
                        }
                        req.Reply(true, nil)
                        // Now read stdin lines and echo marker with exit code 0
                        br := bufio.NewReader(ch)
                        for {
                            line, err := br.ReadString('\n')
                            if err != nil { return }
                            i := strings.LastIndex(line, "echo ")
                            j := strings.LastIndex(line, " $?")
                            if i >= 0 && j > i+5 {
                                marker := strings.TrimSpace(strings.Trim(line[i+5:j], "'"))
                                ch.Write([]byte("INTEGRATION\n"))
                                ch.Write([]byte(marker+" 0\n"))
                            }
                        }
                    default:
                        req.Reply(false, nil)
                    }
                }
            }(c, reqs)
        }
    }()

    // Connect client
    clientCfg := &ssh.ClientConfig{User: "u", HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 3 * time.Second}
    client, err := ssh.Dial("tcp", ln.Addr().String(), clientCfg)
    if err != nil { t.Fatalf("client dial: %v", err) }
    defer client.Close()

    ps, err := newPersistentShell(client)
    if err != nil { t.Fatalf("newPersistentShell: %v", err) }
    out, code, err := ps.runOne("date")
    if err != nil { t.Fatalf("runOne: %v", err) }
    if code != 0 { t.Fatalf("exit code=%d, want 0", code) }
    if string(out) != "INTEGRATION\n" { t.Fatalf("unexpected out: %q", string(out)) }
    _ = ps.Close()

    _ = client.Close()
    _ = ln.Close()
    <-serverDone
}

// Note: Negative PTY/exec cases are avoided to prevent test flakiness.

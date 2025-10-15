package cmd

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "github.com/spf13/pflag"
    "github.com/spf13/viper"
    "github.com/stretchr/testify/require"
    "golang.org/x/crypto/ssh"
)

// writeTemp creates a temp file with content and returns its path.
func writeTemp(t *testing.T, dir, name, content string) string {
    t.Helper()
    p := filepath.Join(dir, name)
    require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
    require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
    return p
}

// resetConfig clears global configuration so tests don't leak state
func resetConfig() {
    viper.Reset()
    viper.SetEnvPrefix("ARBOR_EXFIL")
    viper.AutomaticEnv()
    // Reset flags to defaults and clear Changed status
    rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
        _ = f.Value.Set(f.DefValue)
        f.Changed = false
    })
    cfgManifest = ""
    cfgTarget = ""
    cfgUser = ""
    cfgPassword = ""
    cfgKeyPath = ""
    cfgPassphrase = ""
    cfgOutPath = ""
    cfgKnownHosts = ""
    cfgStrictHost = true
    cfgTimeout = 0
    cfgConnTimeout = 0
}

func TestRootExecute_Success(t *testing.T) {
    resetConfig()
    // Stub SSH dial and command execution
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun })

    dialSSHFunc = func(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
        return nil, nil // nil is fine; code guards Close() and our run stub ignores it
    }

    call := 0
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        call++
        switch call {
        case 1:
            return []byte("out1\n"), 0, nil
        case 2:
            return []byte("out2"), 1, fmt.Errorf("exit status 1")
        default:
            return []byte("unexpected"), 0, nil
        }
    }

    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "manifests/m.yaml", `
name: Test Run
description: Run two commands
commands:
  - command: echo
    args: ["hello world", "a'b"]
  - command: whoami
`)
    outPath := filepath.Join(tmp, "out.txt")

    // Set CLI args
    rootCmd.SetArgs([]string{
        "--target", "127.0.0.1:22",
        "--user", "tester",
        "--manifest", manifestPath,
        "--out", outPath,
        "--strict-host-key=false",
    })

    err := rootCmd.Execute()
    require.NoError(t, err)

    b, err := os.ReadFile(outPath)
    require.NoError(t, err)
    out := string(b)

    // Header
    require.Contains(t, out, "Name: Test Run")
    require.Contains(t, out, "Description: Run two commands")
    // Sections
    require.Contains(t, out, `Command: echo 'hello world' 'a'\''b'`)
    require.Contains(t, out, "Command: whoami\n")
    // Exit codes and outputs
    require.Contains(t, out, "Exit Code: 0")
    require.Contains(t, out, "Exit Code: 1")
    require.Contains(t, out, "Error: exit status 1")
    // Ensures newline appended when missing
    require.Contains(t, out, "out2\n---8<---")
}

func TestRootExecute_DialsOnceForMultipleCommands(t *testing.T) {
    resetConfig()
    // Stub SSH dial and command execution
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun })

    dialCalls := 0
    dialSSHFunc = func(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
        dialCalls++
        return nil, nil
    }

    // Each command should run via a new session on the same client
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        return []byte("ok\n"), 0, nil
    }

    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: cmd1
  - command: cmd2
`)
    outPath := filepath.Join(tmp, "out.txt")

    rootCmd.SetArgs([]string{
        "--target", "127.0.0.1:22",
        "--user", "tester",
        "--manifest", manifestPath,
        "--out", outPath,
        "--strict-host-key=false",
    })

    err := rootCmd.Execute()
    require.NoError(t, err)
    // Ensure exactly one SSH dial for multiple commands
    require.Equal(t, 1, dialCalls)
}

func TestRootExecute_TimeoutReconnect(t *testing.T) {
    resetConfig()
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun })

    dialCalls := 0
    dialSSHFunc = func(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
        dialCalls++
        return nil, nil
    }

    call := 0
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        call++
        if call == 1 {
            return nil, -1, context.DeadlineExceeded
        }
        return []byte("ok\n"), 0, nil
    }

    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: Timeout Test
description: Should reconnect after timeout
commands:
  - command: cmd1
  - command: cmd2
`)
    outPath := filepath.Join(tmp, "out.txt")

    rootCmd.SetArgs([]string{
        "--target", "127.0.0.1:22",
        "--user", "tester",
        "--manifest", manifestPath,
        "--out", outPath,
        "--cmd-timeout", "1s",
        "--strict-host-key=false",
    })
    err := rootCmd.Execute()
    require.NoError(t, err)

    // First initial dial + one reconnect after timeout
    require.Equal(t, 2, dialCalls)

    b, err := os.ReadFile(outPath)
    require.NoError(t, err)
    s := string(b)
    require.Contains(t, s, "Error: context deadline exceeded")
    require.Contains(t, s, "Exit Code: -1")
}

func TestRootExecute_ValidationErrors(t *testing.T) {
    resetConfig()
    // Missing target
    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: x
`)
    rootCmd.SetArgs([]string{
        "--user", "u",
        "--manifest", manifestPath,
        "--out", filepath.Join(tmp, "out.txt"),
        "--strict-host-key=false",
    })
    err := rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "--target is required")

    // Missing manifest
    resetConfig()
    rootCmd.SetArgs([]string{
        "--user", "u",
        "--target", "127.0.0.1:22",
        "--out", filepath.Join(tmp, "out.txt"),
        "--strict-host-key=false",
    })
    err = rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "--manifest is required")

    // Manifest missing description
    resetConfig()
    mfBad := writeTemp(t, tmp, "bad.yaml", `
name: OnlyName
commands:
  - command: x
`)
    rootCmd.SetArgs([]string{
        "--user", "u",
        "--target", "127.0.0.1:22",
        "--manifest", mfBad,
        "--out", filepath.Join(tmp, "out.txt"),
        "--strict-host-key=false",
    })
    err = rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "manifest.description is required")
    // Empty commands
    resetConfig()
    mfEmpty := writeTemp(t, tmp, "empty.yaml", `
name: OnlyName
description: D
commands: []
`)
    rootCmd.SetArgs([]string{
        "--user", "u",
        "--target", "127.0.0.1:22",
        "--manifest", mfEmpty,
        "--out", filepath.Join(tmp, "out.txt"),
        "--strict-host-key=false",
    })
    err = rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "manifest contains no commands")
}

func TestShellQuote(t *testing.T) {
    require.Equal(t, "simple", shellQuote("simple"))
    require.Equal(t, "''", shellQuote(""))
    require.Equal(t, "'two words'", shellQuote("two words"))
    require.Equal(t, `"'a'\''b'"`[1:len(`"'a'\''b'"`)-1], shellQuote("a'b"))
    require.Equal(t, "/path/ok", shellQuote("/path/ok"))
    require.Equal(t, "abc+123", shellQuote("abc+123"))
}

func TestWriteHeaderAndSection(t *testing.T) {
    var buf bytes.Buffer
    mf := &manifest{Name: "N", Description: "D", Commands: []commandEntry{{Command: "x"}}}
    writeHeader(&buf, mf)
    s := buf.String()
    require.Contains(t, s, "Name: N")
    require.Contains(t, s, "Description: D")
    require.Contains(t, s, "Command Count: 1")

    buf.Reset()
    err := writeCommandSection(&buf, commandEntry{Command: "x"}, []byte("hello\n"), 0, nil, 0)
    require.NoError(t, err)
    out := buf.String()
    require.Contains(t, out, "Command: x\n")
    require.Contains(t, out, "Exit Code: 0")
    require.Contains(t, out, "---8<---\nhello\n---8<---")

    buf.Reset()
    err = writeCommandSection(&buf, commandEntry{Command: "x"}, []byte("no-nl"), 2, errors.New("boom"), 3*time.Second)
    require.NoError(t, err)
    out = buf.String()
    require.Contains(t, out, "Timeout: 3s")
    require.Contains(t, out, "Exit Code: 2")
    require.Contains(t, out, "Error: boom")
    require.True(t, strings.Contains(out, "no-nl\n---8<---"))
}

func TestPerCommandTimeout(t *testing.T) {
    var c commandEntry
    def := 10 * time.Second
    require.Equal(t, def, c.perCommandTimeout(def))
    c.Timeout = "bad"
    require.Equal(t, def, c.perCommandTimeout(def))
    c.Timeout = "1s"
    require.Equal(t, 1*time.Second, c.perCommandTimeout(def))
}

func TestDialSSH_StrictHostKeyMissingKnownHosts(t *testing.T) {
    // Ensure we fail closed when strict host key and no known_hosts file.
    _, err := dialSSH("127.0.0.1:22", "u", "", "", "", filepath.Join(t.TempDir(), "nope"), true, 100*time.Millisecond)
    require.Error(t, err)
    require.Contains(t, err.Error(), "known_hosts file not found")
}

func TestLoadManifest_AndAliasCmd(t *testing.T) {
    tmp := t.TempDir()
    p := writeTemp(t, tmp, "m.yaml", `
name: A
description: B
commands:
  - cmd: show version
    args: ["arg1", "arg 2"]
`)
    mf, err := loadManifest(p)
    require.NoError(t, err)
    require.Equal(t, "A", mf.Name)
    require.Equal(t, 1, len(mf.Commands))
    require.Equal(t, "show version", mf.Commands[0].Command)
    // Ensure line builds with quoting
    require.Equal(t, "show version arg1 'arg 2'", mf.Commands[0].line())
}

func TestEnvOverrides_Initialize(t *testing.T) {
    resetConfig()
    t.Setenv("ARBOR_EXFIL_PASSWORD", "secret")
    t.Setenv("ARBOR_EXFIL_PASSPHRASE", "pp")

    tmp := t.TempDir()
    // Intentionally provide insufficient args to force validation error after init
    rootCmd.SetArgs([]string{
        "--manifest", filepath.Join(tmp, "missing.yaml"),
        "--out", filepath.Join(tmp, "out.txt"),
    })
    _ = rootCmd.Execute()
    require.Equal(t, "secret", cfgPassword)
    require.Equal(t, "pp", cfgPassphrase)
}

func TestAdminUserRejected_ErrorReturned(t *testing.T) {
    resetConfig()
    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: x
`)
    outPath := filepath.Join(tmp, "out.txt")
    rootCmd.SetArgs([]string{
        "--target", "127.0.0.1:22",
        "--user", "admin",
        "--manifest", manifestPath,
        "--out", outPath,
        "--strict-host-key=false",
    })
    err := rootCmd.Execute()
    require.Error(t, err)
    require.True(t, errors.Is(err, errAdminUser))
}

func TestExecute_AdminUser_PrintsStdoutAndExit1(t *testing.T) {
    resetConfig()
    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: x
`)
    outPath := filepath.Join(tmp, "out.txt")
    rootCmd.SetArgs([]string{
        "--target", "127.0.0.1:22",
        "--user", "admin",
        "--manifest", manifestPath,
        "--out", outPath,
        "--strict-host-key=false",
    })

    // Capture stdout
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w
    defer func() { os.Stdout = oldStdout }()

    // Capture exit code via stub
    code := 0
    origExit := exitFunc
    exitFunc = func(c int) { code = c }
    defer func() { exitFunc = origExit }()

    // Call wrapper Execute (not rootCmd.Execute) so it handles admin specially
    Execute()

    // Read stdout
    _ = w.Close()
    var buf bytes.Buffer
    _, _ = buf.ReadFrom(r)
    out := buf.String()
    require.Contains(t, out, "admin user account cannot be used")
    require.Equal(t, 1, code)
}

func TestRootExecute_OutputDirCreationError(t *testing.T) {
    resetConfig()
    // Create a file where a directory is expected
    tmp := t.TempDir()
    badDir := writeTemp(t, tmp, "notadir", "content")

    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: x
`)
    outPath := filepath.Join(badDir, "out.txt")

    rootCmd.SetArgs([]string{
        "--target", "127.0.0.1:22",
        "--user", "tester",
        "--manifest", manifestPath,
        "--out", outPath,
        "--strict-host-key=false",
    })
    err := rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "failed to create output dir")
}

func TestDialSSH_StrictHostKeyWithKnownHosts(t *testing.T) {
    // With a present known_hosts file, we should pass host key setup and then
    // likely fail on connect (which is fine for coverage).
    tmp := t.TempDir()
    kh := writeTemp(t, tmp, "known_hosts", "\n")
    _, err := dialSSH("127.0.0.1:1", "u", "", "", "", kh, true, 50*time.Millisecond)
    require.Error(t, err)
}

func TestLoadSigner_FileNotFound(t *testing.T) {
    _, err := loadSigner(filepath.Join(t.TempDir(), "missing_key"), "")
    require.Error(t, err)
}

func TestLoadSigner_RSAKey_Success(t *testing.T) {
    tmp := t.TempDir()
    // Generate a small RSA key for testing
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    require.NoError(t, err)
    b := x509.MarshalPKCS1PrivateKey(key)
    pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b})
    p := writeTemp(t, tmp, "id_rsa", string(pemBytes))
    s, err := loadSigner(p, "")
    require.NoError(t, err)
    require.NotNil(t, s.PublicKey())
}

func TestDialSSH_AuthMethodsAssembly(t *testing.T) {
    // Set SSH_AUTH_SOCK to a non-existent path to exercise agent branch
    t.Setenv("SSH_AUTH_SOCK", filepath.Join(t.TempDir(), "no.sock"))
    // Generate a key to exercise public key auth loading
    tmp := t.TempDir()
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    require.NoError(t, err)
    b := x509.MarshalPKCS1PrivateKey(key)
    pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b})
    keyPath := writeTemp(t, tmp, "id_rsa", string(pemBytes))

    // Use strictHost=false to avoid known_hosts requirements; expect dial error
    _, err = dialSSH("127.0.0.1:1", "u", "p", keyPath, "", "", false, 50*time.Millisecond)
    require.Error(t, err)
}

// Fake session and client for runRemoteCommand tests
type fakeSession struct {
    out     []byte
    err     error
    delay   time.Duration
    closed  bool
}

func (f *fakeSession) CombinedOutput(cmd string) ([]byte, error) {
    if f.delay > 0 { time.Sleep(f.delay) }
    return f.out, f.err
}
func (f *fakeSession) Close() error { f.closed = true; return nil }

type fakeClient struct{ sess *fakeSession; newErr error }

func (c *fakeClient) NewSession() (session, error) {
    if c.newErr != nil { return nil, c.newErr }
    return c.sess, nil
}

func TestRunRemoteCommand_Success(t *testing.T) {
    s := &fakeSession{out: []byte("OK\n"), err: nil}
    out, code, err := runRemoteCommand(&fakeClient{sess: s}, "echo OK", 0)
    require.NoError(t, err)
    require.Equal(t, 0, code)
    require.Equal(t, "OK\n", string(out))
    require.True(t, s.closed)
}

func TestRunRemoteCommand_Timeout(t *testing.T) {
    s := &fakeSession{out: []byte("SLOW\n"), delay: 200 * time.Millisecond}
    out, code, err := runRemoteCommand(&fakeClient{sess: s}, "sleep", 10*time.Millisecond)
    require.Error(t, err)
    require.True(t, errors.Is(err, context.DeadlineExceeded))
    require.Equal(t, -1, code)
    require.Nil(t, out)
}

func TestRunRemoteCommand_NewSessionError(t *testing.T) {
    out, code, err := runRemoteCommand(&fakeClient{newErr: errors.New("no session")}, "cmd", 0)
    require.Error(t, err)
    require.Equal(t, -1, code)
    require.Nil(t, out)
}

func TestRunRemoteCommand_CommandError_NoExitCode(t *testing.T) {
    s := &fakeSession{out: []byte("oops\n"), err: errors.New("boom")}
    out, code, err := runRemoteCommand(&fakeClient{sess: s}, "cmd", 0)
    require.Error(t, err)
    require.Equal(t, -1, code)
    require.Equal(t, "oops\n", string(out))
}

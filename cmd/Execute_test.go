package cmd

import (
    "bytes"
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "golang.org/x/crypto/ssh"
)

// Happy path: Execute() should not call exitFunc when rootCmd succeeds.
func TestExecute_Success_NoExit(t *testing.T) {
    resetConfig()

    // Stub SSH dial and remote command to avoid network
    origDial := dialSSHFunc
    origRun := runRemoteCommandFunc
    origExit := exitFunc
    t.Cleanup(func() { dialSSHFunc = origDial; runRemoteCommandFunc = origRun; exitFunc = origExit })

    dialSSHFunc = func(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
        return nil, nil
    }
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        return []byte("ok\n"), 0, nil
    }

    calledExit := 0
    exitFunc = func(code int) { calledExit = code }

    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: echo
    shell: /bin/sh
    args: ["ok"]
`)
    outPath := filepath.Join(tmp, "out.txt")

    rootCmd.SetArgs([]string{
        "run",
        "--target", "127.0.0.1:22",
        "--user", "tester",
        "--manifest", manifestPath,
        "--out", outPath,
        "--strict-host-key=false",
    })

    // Call wrapper
    Execute()

    // Expect no exit
    require.Equal(t, 0, calledExit)
    // And output file created by root command
    _, err := os.Stat(outPath)
    require.NoError(t, err)
}

// Generic error path: Execute() should print to stderr and call exit 1 when
// root command returns a non-admin error (e.g., missing required flags).
func TestExecute_GenericError_Exit1(t *testing.T) {
    resetConfig()

    // Set args missing target to force validation error
    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: x
    shell: /bin/sh
`)
    rootCmd.SetArgs([]string{
        "run",
        "--user", "tester",
        "--manifest", manifestPath,
        "--out", filepath.Join(tmp, "out.txt"),
        "--strict-host-key=false",
    })

    // Capture stderr
    oldStderr := os.Stderr
    r, w, _ := os.Pipe()
    os.Stderr = w
    defer func() { os.Stderr = oldStderr }()

    // Stub exit
    code := 0
    origExit := exitFunc
    exitFunc = func(c int) { code = c }
    defer func() { exitFunc = origExit }()

    Execute()

    _ = w.Close()
    var buf bytes.Buffer
    _, _ = buf.ReadFrom(r)
    errOut := buf.String()
    require.Contains(t, errOut, "--target is required")
    require.Equal(t, 1, code)
}

// Sad path: admin user should print to stdout and call exit 1
func TestExecute_AdminUser_StdoutExit1_Dedicated(t *testing.T) {
    resetConfig()

    tmp := t.TempDir()
    manifestPath := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: x
    shell: /bin/sh
`)
    outPath := filepath.Join(tmp, "out.txt")
    rootCmd.SetArgs([]string{
        "run",
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

    // Stub exit
    code := 0
    origExit := exitFunc
    exitFunc = func(c int) { code = c }
    defer func() { exitFunc = origExit }()

    Execute()

    // Read stdout
    _ = w.Close()
    var buf bytes.Buffer
    _, _ = buf.ReadFrom(r)
    out := buf.String()
    require.Contains(t, out, "admin user account cannot be used")
    require.Equal(t, 1, code)
}

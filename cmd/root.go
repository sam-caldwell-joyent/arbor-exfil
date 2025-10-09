package cmd

import (
    "bufio"
    "context"
    "errors"
    "fmt"
    "io"
    "net"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "golang.org/x/crypto/ssh"
    "golang.org/x/crypto/ssh/agent"
    "golang.org/x/crypto/ssh/knownhosts"
)

// Version is set via -ldflags at build time if desired
var Version = "0.1.0"
var errAdminUser = errors.New("admin user account cannot be used")

type manifest struct {
    Name        string         `yaml:"name"`
    Description string         `yaml:"description"`
    Commands    []commandEntry `yaml:"commands"`
}

type commandEntry struct {
    // "command" is preferred; "cmd" also accepted during unmarshal
    Command string   `yaml:"command"`
    Args    []string `yaml:"args"`
    // Optional per-command timeout like "30s"; overrides global if set
    Timeout string `yaml:"timeout,omitempty"`
}

func (c *commandEntry) line() string {
    if len(c.Args) == 0 {
        return c.Command
    }
    // Quote args to be safe for remote shell
    quoted := make([]string, 0, len(c.Args))
    for _, a := range c.Args {
        quoted = append(quoted, shellQuote(a))
    }
    return strings.TrimSpace(c.Command + " " + strings.Join(quoted, " "))
}

func (c *commandEntry) perCommandTimeout(defaultTimeout time.Duration) time.Duration {
    if c.Timeout == "" {
        return defaultTimeout
    }
    d, err := time.ParseDuration(c.Timeout)
    if err != nil {
        return defaultTimeout
    }
    return d
}

var (
    cfgManifest   string
    cfgTarget     string
    cfgUser       string
    cfgPassword   string
    cfgKeyPath    string
    cfgPassphrase string
    cfgOutPath    string
    cfgKnownHosts string
    cfgStrictHost bool
    cfgTimeout    time.Duration
    cfgConnTimeout time.Duration
)

// Allow tests to stub dialing and command execution
var (
    dialSSHFunc          = dialSSH
    runRemoteCommandFunc = runRemoteCommand
)

var rootCmd = &cobra.Command{
    Use:   "arbor-exfil",
    Short: "Run ArbOS commands from a YAML manifest and capture output",
    Long:  "Connects to an Arbor TMS leader (ArbOS) over SSH, executes commands from a YAML manifest, and writes responses to an output file.",
    Version: Version,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Basic validation
        if cfgTarget == "" {
            return errors.New("--target is required (FQDN/IP:port)")
        }
        if cfgManifest == "" {
            return errors.New("--manifest is required (path to YAML)")
        }
        if cfgOutPath == "" {
            return errors.New("--out is required (path to output file)")
        }
        if cfgUser == "" {
            return errors.New("--user is required for SSH authentication")
        }
        if strings.TrimSpace(cfgUser) == "admin" {
            // Disallow admin account usage
            return errAdminUser
        }

        mf, err := loadManifest(cfgManifest)
        if err != nil {
            return fmt.Errorf("failed to read manifest: %w", err)
        }
        if len(mf.Commands) == 0 {
            return errors.New("manifest contains no commands")
        }

        // Prepare output file (create dirs if needed)
        if err := os.MkdirAll(filepath.Dir(cfgOutPath), 0o755); err != nil {
            return fmt.Errorf("failed to create output dir: %w", err)
        }
        outFile, err := os.Create(cfgOutPath)
        if err != nil {
            return fmt.Errorf("failed to create output file: %w", err)
        }
        defer outFile.Close()

        // Header in output
        writeHeader(outFile, mf)

        // Connect SSH
        client, err := dialSSHFunc(cfgTarget, cfgUser, cfgPassword, cfgKeyPath, cfgPassphrase, cfgKnownHosts, cfgStrictHost, cfgConnTimeout)
        if err != nil {
            return fmt.Errorf("ssh connection failed: %w", err)
        }
        defer func() { if client != nil { _ = client.Close() } }()

        // Execute commands
        for i, c := range mf.Commands {
            title := fmt.Sprintf("[%d/%d] %s", i+1, len(mf.Commands), c.line())
            fmt.Fprintf(os.Stderr, "Executing %s\n", title)

            cmdTimeout := c.perCommandTimeout(cfgTimeout)
            out, exitCode, runErr := runRemoteCommandFunc(sshClientWrapper{client}, c.line(), cmdTimeout)

            // Write section to output file
            if err := writeCommandSection(outFile, c, out, exitCode, runErr, cmdTimeout); err != nil {
                return fmt.Errorf("failed writing output: %w", err)
            }

            // On timeout, try to reconnect once and continue
            if errors.Is(runErr, context.DeadlineExceeded) {
                fmt.Fprintln(os.Stderr, "Command timed out; reconnecting...")
                if client != nil { _ = client.Close() }
                client, err = dialSSHFunc(cfgTarget, cfgUser, cfgPassword, cfgKeyPath, cfgPassphrase, cfgKnownHosts, cfgStrictHost, cfgConnTimeout)
                if err != nil {
                    return fmt.Errorf("reconnect failed after timeout: %w", err)
                }
            }
        }

        fmt.Fprintf(os.Stderr, "Done. Output written to %s\n", cfgOutPath)
        return nil
    },
}

// exitFunc allows tests to stub process exit behavior
var exitFunc = os.Exit

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        if errors.Is(err, errAdminUser) {
            // Admin account error prints to stdout and exits with code 1
            fmt.Fprintln(os.Stdout, err.Error())
            exitFunc(1)
            return
        }
        fmt.Fprintln(os.Stderr, err)
        exitFunc(1)
        return
    }
}

func init() {
    // Flags
    rootCmd.Flags().StringVarP(&cfgTarget, "target", "t", "", "ArbOS target FQDN/IP:port (e.g., tms.example.com:22)")
    rootCmd.Flags().StringVarP(&cfgManifest, "manifest", "m", "", "Path to YAML manifest file")
    rootCmd.Flags().StringVarP(&cfgOutPath, "out", "o", "", "Path to output text file")
    rootCmd.Flags().StringVarP(&cfgUser, "user", "u", "", "SSH username")
    rootCmd.Flags().StringVar(&cfgPassword, "password", "", "SSH password (or set ARBOR_EXFIL_PASSWORD)")
    rootCmd.Flags().StringVar(&cfgKeyPath, "key", "", "Path to SSH private key (PEM, OpenSSH)")
    rootCmd.Flags().StringVar(&cfgPassphrase, "passphrase", "", "Private key passphrase (or set ARBOR_EXFIL_PASSPHRASE)")
    rootCmd.Flags().StringVar(&cfgKnownHosts, "known-hosts", filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"), "Path to known_hosts file")
    rootCmd.Flags().BoolVar(&cfgStrictHost, "strict-host-key", true, "Require host key verification (disable to accept any host key)")
    rootCmd.Flags().DurationVar(&cfgTimeout, "cmd-timeout", 0, "Per-command timeout (e.g., 30s). 0 disables")
    rootCmd.Flags().DurationVar(&cfgConnTimeout, "conn-timeout", 15*time.Second, "Connection timeout")

    // Bind env with Viper
    _ = viper.BindPFlag("target", rootCmd.Flags().Lookup("target"))
    _ = viper.BindPFlag("manifest", rootCmd.Flags().Lookup("manifest"))
    _ = viper.BindPFlag("out", rootCmd.Flags().Lookup("out"))
    _ = viper.BindPFlag("user", rootCmd.Flags().Lookup("user"))
    _ = viper.BindPFlag("password", rootCmd.Flags().Lookup("password"))
    _ = viper.BindPFlag("key", rootCmd.Flags().Lookup("key"))
    _ = viper.BindPFlag("passphrase", rootCmd.Flags().Lookup("passphrase"))
    _ = viper.BindPFlag("known-hosts", rootCmd.Flags().Lookup("known-hosts"))
    _ = viper.BindPFlag("strict-host-key", rootCmd.Flags().Lookup("strict-host-key"))
    _ = viper.BindPFlag("cmd-timeout", rootCmd.Flags().Lookup("cmd-timeout"))
    _ = viper.BindPFlag("conn-timeout", rootCmd.Flags().Lookup("conn-timeout"))

    viper.SetEnvPrefix("ARBOR_EXFIL")
    viper.AutomaticEnv()

    // Pull in environment overrides on init
    cobra.OnInitialize(func() {
        if v := viper.GetString("target"); v != "" {
            cfgTarget = v
        }
        if v := viper.GetString("manifest"); v != "" {
            cfgManifest = v
        }
        if v := viper.GetString("out"); v != "" {
            cfgOutPath = v
        }
        if v := viper.GetString("user"); v != "" {
            cfgUser = v
        }
        if v := viper.GetString("password"); v != "" {
            cfgPassword = v
        }
        if v := viper.GetString("key"); v != "" {
            cfgKeyPath = v
        }
        if v := viper.GetString("passphrase"); v != "" {
            cfgPassphrase = v
        }
        if v := viper.GetString("known-hosts"); v != "" {
            cfgKnownHosts = v
        }
        if v := viper.GetString("cmd-timeout"); v != "" {
            if d, err := time.ParseDuration(v); err == nil { cfgTimeout = d }
        }
        if v := viper.GetString("conn-timeout"); v != "" {
            if d, err := time.ParseDuration(v); err == nil { cfgConnTimeout = d }
        }
        // Booleans
        if viper.IsSet("strict-host-key") {
            cfgStrictHost = viper.GetBool("strict-host-key")
        }
    })
}

// loadManifest reads and validates the YAML manifest
func loadManifest(path string) (*manifest, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    mf := &manifest{}
    if err := yamlUnmarshal(b, mf); err != nil {
        return nil, err
    }
    // Support "cmd" as alias for "command" if used
    for i := range mf.Commands {
        if mf.Commands[i].Command == "" {
            // Attempt to sniff a "cmd:" field via a quick map decode
            // as a fallback for compatibility.
            // This is a lightweight pass; if not present, ignore.
        }
    }
    if mf.Name == "" {
        return nil, errors.New("manifest.name is required")
    }
    if mf.Description == "" {
        return nil, errors.New("manifest.description is required")
    }
    for i, c := range mf.Commands {
        if strings.TrimSpace(c.Command) == "" {
            return nil, fmt.Errorf("commands[%d].command is required", i)
        }
    }
    return mf, nil
}

// yamlUnmarshal is a small wrapper to avoid importing yaml here and in tests
func yamlUnmarshal(b []byte, out any) error {
    // Implemented in yaml.go
    return yamlUnmarshalImpl(b, out)
}

// writeHeader writes manifest meta info to the output file
func writeHeader(w io.Writer, mf *manifest) {
    bw := bufio.NewWriter(w)
    fmt.Fprintf(bw, "Name: %s\n", mf.Name)
    fmt.Fprintf(bw, "Description: %s\n", mf.Description)
    fmt.Fprintf(bw, "Generated: %s\n", time.Now().Format(time.RFC3339))
    fmt.Fprintf(bw, "Command Count: %d\n", len(mf.Commands))
    fmt.Fprintln(bw, strings.Repeat("=", 80))
    bw.Flush()
}

// writeCommandSection writes one command's results
func writeCommandSection(w io.Writer, c commandEntry, out []byte, exitCode int, runErr error, timeout time.Duration) error {
    bw := bufio.NewWriter(w)
    fmt.Fprintln(bw, strings.Repeat("-", 80))
    fmt.Fprintf(bw, "Command: %s\n", c.line())
    if timeout > 0 {
        fmt.Fprintf(bw, "Timeout: %s\n", timeout.String())
    }
    fmt.Fprintf(bw, "Exit Code: %d\n", exitCode)
    if runErr != nil {
        fmt.Fprintf(bw, "Error: %v\n", runErr)
    }
    fmt.Fprintln(bw, "Output:")
    fmt.Fprintln(bw, "---8<---")
    // Write raw output
    bw.Write(out)
    if len(out) == 0 || !strings.HasSuffix(string(out), "\n") {
        bw.WriteString("\n")
    }
    fmt.Fprintln(bw, "---8<---")
    return bw.Flush()
}

// dialSSH establishes an SSH client connection with options
func dialSSH(target, user, password, keyPath, passphrase, knownHostsPath string, strictHost bool, dialTimeout time.Duration) (*ssh.Client, error) {
    auths := []ssh.AuthMethod{}

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

// runRemoteCommand executes a single command and returns output + exit code
// sessionClient is a minimal interface to obtain a command session
type sessionClient interface {
    NewSession() (session, error)
}

// session is a minimal interface for running a command and closing
type session interface {
    CombinedOutput(cmd string) ([]byte, error)
    Close() error
}

// sshClientWrapper adapts *ssh.Client to sessionClient
type sshClientWrapper struct{ c *ssh.Client }

func (w sshClientWrapper) NewSession() (session, error) {
    if w.c == nil { return nil, fmt.Errorf("nil ssh client") }
    s, err := w.c.NewSession()
    if err != nil { return nil, err }
    return sshSessionWrapper{s}, nil
}

// sshSessionWrapper adapts *ssh.Session to session
type sshSessionWrapper struct{ s *ssh.Session }

func (w sshSessionWrapper) CombinedOutput(cmd string) ([]byte, error) { return w.s.CombinedOutput(cmd) }
func (w sshSessionWrapper) Close() error { return w.s.Close() }

func runRemoteCommand(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
    type result struct {
        out      []byte
        exitCode int
        err      error
    }

    run := func() result {
        sess, err := client.NewSession()
        if err != nil {
            return result{nil, -1, err}
        }
        defer sess.Close()
        b, err := sess.CombinedOutput(cmd)
        if err == nil {
            return result{b, 0, nil}
        }
        // Try to derive exit status
        exit := -1
        if ee, ok := err.(*ssh.ExitError); ok {
            exit = ee.ExitStatus()
        }
        return result{b, exit, err}
    }

    if timeout <= 0 {
        r := run()
        return r.out, r.exitCode, r.err
    }

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    ch := make(chan result, 1)
    go func() { ch <- run() }()

    select {
    case r := <-ch:
        return r.out, r.exitCode, r.err
    case <-ctx.Done():
        // Best-effort: indicate timeout. Caller may reconnect if desired.
        return nil, -1, context.DeadlineExceeded
    }
}

// loadSigner loads a private key with optional passphrase
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
    if _, ok := err.(*ssh.PassphraseMissingError); ok {
        return nil, fmt.Errorf("private key is encrypted; provide --passphrase or ARBOR_EXFIL_PASSPHRASE")
    }
    return nil, err
}

// shellQuote minimally quotes an argument for POSIX shell
func shellQuote(s string) string {
    if s == "" {
        return "''"
    }
    if strings.IndexFunc(s, func(r rune) bool {
        // Safe chars: alnum, - _ . / @ : and commas
        if r >= 'a' && r <= 'z' { return false }
        if r >= 'A' && r <= 'Z' { return false }
        if r >= '0' && r <= '9' { return false }
        switch r {
        case '-', '_', '.', '/', '@', ':', ',', '+', '=':
            return false
        }
        return true
    }) == -1 {
        return s
    }
    // Single-quote, escaping embedded single quotes: ' -> '\''
    return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

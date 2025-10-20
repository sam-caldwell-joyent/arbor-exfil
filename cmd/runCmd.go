package cmd

import (
    "bufio"
    "context"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
)

// runCmd executes the primary workflow: connects to the leader, performs host
// discovery, and when commands are present executes them across all discovered
// hosts using a single persistent PTY shell per connection. Results are written
// to a structured YAML report. With --noop, it only prints planned commands.
var runCmd = &cobra.Command{
    Use:   "run",
    Short: "Execute manifest-driven collection run",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Require manifest and output path first, since manifest may provide defaults
        if cfgManifest == "" {
            return errors.New("--manifest is required (path to YAML)")
        }
        if cfgOutPath == "" {
            return errors.New("--out is required (path to output file)")
        }

        // Load manifest to potentially source ssh_host defaults
        mf, err := loadManifest(cfgManifest)
        if err != nil {
            return fmt.Errorf("failed to read manifest: %w", err)
        }
        // If no commands are specified, we still perform host discovery and
        // write only the discovered hosts list to the output.

        // Fallback to manifest ssh_host when CLI args are not provided
        if cfgTarget == "" {
            if host := strings.TrimSpace(mf.SSHHost.IP); host != "" {
                if strings.Contains(host, ":") {
                    cfgTarget = host
                } else {
                    cfgTarget = host + ":22"
                }
            }
        }
        if cfgUser == "" {
            if u := strings.TrimSpace(mf.SSHHost.User); u != "" {
                cfgUser = u
            }
        }

        // Remaining basic validation after applying manifest defaults
        if cfgTarget == "" {
            return errors.New("--target is required (FQDN/IP:port)")
        }
        if cfgUser == "" {
            return errors.New("--user is required for SSH authentication")
        }
        if adminUser := strings.TrimSpace(cfgUser); adminUser == "admin" || adminUser == "root" {
            // Disallow admin account usage
            return errAdminUser
        }

        // Require a shell for each command; no implicit fallback
        if len(mf.Commands) > 0 {
            for i, c := range mf.Commands {
                if strings.TrimSpace(c.Shell) == "" {
                    return fmt.Errorf("commands[%d].shell is required", i)
                }
            }
        }

        // Prepare output file (create dirs if needed)
        if err := os.MkdirAll(filepath.Dir(cfgOutPath), 0o755); err != nil {
            return fmt.Errorf("failed to create output dir: %w", err)
        }
        outFile, err := os.Create(cfgOutPath)
        if err != nil {
            return fmt.Errorf("failed to create output file: %w", err)
        }
        defer func(outFile *os.File) {
            err := outFile.Close()
            if err != nil {
                panic(err)
            }
        }(outFile)

        // Prepare YAML report model
        report := newYAMLReport(mf)

        // Connect SSH
        client, err := dialSSHFunc(cfgTarget, cfgUser, cfgPassword, cfgKeyPath, cfgPassphrase, cfgKnownHosts,
            cfgStrictHost, cfgConnTimeout)

        if err != nil {
            return fmt.Errorf("ssh connection failed: %w", err)
        }
        defer func() {
            if client != nil {
                _ = client.Close()
            }
        }()

        // Establish one persistent session over the SSH connection (if client is non-nil).
        // In unit tests, dial may be stubbed to return nil; in that case, fall back
        // to the wrapper and the test's run stub will drive behavior.
        var (
            ps         *persistentShell
            sessClient sessionClient
        )
        if client != nil {
            ps, err = newPersistentShell(client)
            if err != nil {
                return fmt.Errorf("failed to start persistent session: %w", err)
            }
            defer func() { _ = ps.Close() }()
            sessClient = persistentSessionClient{ps}
        } else {
            sessClient = sshClientWrapper{client}
        }

        // Host discovery
        discOut, _, _, discErr := discoverChildHosts(sessClient, cfgTimeout)
        // Populate discovery section. Include full hosts content when commands exist.
        discovered := []string{}
        if discErr == nil {
            discovered = parseHostsIPs(discOut)
        }
        report.setDiscovery(discOut, discovered, len(mf.Commands) > 0)
        // Parse child hosts (already done into discovered). If no commands, emit YAML with discovery only.
        if len(mf.Commands) == 0 {
            if err := writeYAMLReport(outFile, report); err != nil {
                return fmt.Errorf("failed to write YAML report: %w", err)
            }
            _, _ = fmt.Fprintf(os.Stderr, "Done. Output written to %s\n", cfgOutPath)
            return nil
        }

        // In noop mode, just write planned command strings to debug.out,
        // and still emit a YAML report with discovery only.
        if cfgNoop {
            dbgPath := "debug.out"
            f, err := os.Create(dbgPath)
            if err != nil {
                return fmt.Errorf("create %s: %w", dbgPath, err)
            }
            {
                bw := bufio.NewWriter(f)
                _, _ = fmt.Fprintf(bw, "# Planned commands (%d commands)\n", len(mf.Commands))
                _, _ = fmt.Fprintln(bw, "# Discovery: cat /etc/hosts")
                for _, c := range mf.Commands {
                    shell := c.Shell
                    cmdArg := shellQuote(c.line())
                    if !strings.HasPrefix(cmdArg, "'") || !strings.HasSuffix(cmdArg, "'") {
                        cmdArg = "'" + cmdArg + "'"
                    }
                    full := fmt.Sprintf("sudo -u admin --shell %s -c %s", shellQuote(shell), cmdArg)
                    _, _ = fmt.Fprintln(bw, full)
                }
                _ = bw.Flush()
            }
            _ = f.Close()
            if err := writeYAMLReport(outFile, report); err != nil {
                return fmt.Errorf("failed to write YAML report: %w", err)
            }
            _, _ = fmt.Fprintf(os.Stderr, "Noop mode: wrote planned commands to %s\n", dbgPath)
            _, _ = fmt.Fprintf(os.Stderr, "Done. Output written to %s\n", cfgOutPath)
            return nil
        }

        // Execute commands for each child host
        childHosts := discovered
        if len(childHosts) == 0 {
            childHosts = []string{""}
        }
        for _, host := range childHosts {
            for i, c := range mf.Commands {
                title := fmt.Sprintf("[%d/%d] %s", i+1, len(mf.Commands), c.line())
                _, _ = fmt.Fprintf(os.Stderr, "Executing %s\n", title)
                shell := c.Shell
                cmdArg := shellQuote(c.line())
                if !strings.HasPrefix(cmdArg, "'") || !strings.HasSuffix(cmdArg, "'") {
                    cmdArg = "'" + cmdArg + "'"
                }
                fullCmd := fmt.Sprintf("sudo -u admin --shell %s -c %s", shellQuote(shell), cmdArg)

                cmdTimeout := c.perCommandTimeout(cfgTimeout)
                out, exitCode, runErr := runRemoteCommandFunc(sessClient, fullCmd, cmdTimeout)

                // Accumulate result in report
                res := yamlCmdResult{
                    Title:    strings.TrimSpace(c.Title),
                    Command:  c.line(),
                    Shell:    c.Shell,
                    ExitCode: exitCode,
                    Output:   string(out),
                }
                if cmdTimeout > 0 {
                    res.Timeout = cmdTimeout.String()
                }
                if runErr != nil {
                    res.Error = runErr.Error()
                }
                report.addResult(host, res)

                // On timeout, try to reconnect once and continue
                if errors.Is(runErr, context.DeadlineExceeded) {
                    _, _ = fmt.Fprintln(os.Stderr, "Command timed out; reconnecting...")
                    if ps != nil { _ = ps.Close() }
                    if client != nil { _ = client.Close() }
                    client, err = dialSSHFunc(cfgTarget, cfgUser, cfgPassword, cfgKeyPath, cfgPassphrase,
                        cfgKnownHosts, cfgStrictHost, cfgConnTimeout)
                    if err != nil { return fmt.Errorf("reconnect failed after timeout: %w", err) }
                    // Recreate session client after reconnect
                    if client != nil {
                        ps, err = newPersistentShell(client)
                        if err != nil { return fmt.Errorf("failed to start persistent session after reconnect: %w", err) }
                        sessClient = persistentSessionClient{ps}
                    } else {
                        sessClient = sshClientWrapper{client}
                    }
                }
            }
        }

        // Emit YAML report
        if err := writeYAMLReport(outFile, report); err != nil {
            return fmt.Errorf("failed to write YAML report: %w", err)
        }

        _, _ = fmt.Fprintf(os.Stderr, "Done. Output written to %s\n", cfgOutPath)
        return nil
    },
}

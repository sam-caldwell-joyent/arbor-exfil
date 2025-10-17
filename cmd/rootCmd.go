package cmd

import (
    "context"
    "bufio"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "arbor-exfil",
	Short: "Run ArbOS commands from a YAML manifest and capture output",
	Long: "Connects to an Arbor TMS leader (ArbOS) over SSH, executes commands from a YAML manifest, and writes " +
		"responses to an output file.",
	Version: Version,
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

        // Header in output
        writeHeader(outFile, mf)

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
        discOut, discExit, _, discErr := discoverChildHosts(sessClient, cfgTimeout)
        // If there are commands to run, include discovery section in the report.
        if len(mf.Commands) > 0 {
            discEntry := commandEntry{Command: "cat", Args: []string{"/etc/hosts"}, Title: "Host Discovery (/etc/hosts)"}
            // Pre-write the discovery title and a separator line
            {
                bw := bufio.NewWriter(outFile)
                _, _ = fmt.Fprintln(bw, strings.Repeat("-", 80))
                _, _ = fmt.Fprintln(bw, "Title: "+discEntry.Title)
                _ = bw.Flush()
            }
            if err := writeCommandSection(outFile, discEntry, discOut, discExit, discErr, cfgTimeout); err != nil {
                return fmt.Errorf("failed writing discovery output: %w", err)
            }
        }
        // Parse child hosts if discovery succeeded; on error, proceed with no children.
        var childHosts []string
        if discErr == nil {
            childHosts = parseHostsIPs(discOut)
        }

        // If there are no commands, emit only the discovered hosts list and exit.
        if len(mf.Commands) == 0 {
            bw := bufio.NewWriter(outFile)
            _, _ = fmt.Fprintln(bw, strings.Repeat("-", 80))
            _, _ = fmt.Fprintln(bw, "Discovered Hosts:")
            for _, ip := range childHosts {
                _, _ = fmt.Fprintln(bw, ip)
            }
            _ = bw.Flush()
            _, _ = fmt.Fprintf(os.Stderr, "Done. Output written to %s\n", cfgOutPath)
            return nil
        }

        if len(childHosts) == 0 {
            // Run commands once if no hosts discovered.
            childHosts = []string{""}
        }

        // In noop mode, just write planned command strings to debug.out
        if cfgNoop {
            dbgPath := "debug.out"
            f, err := os.Create(dbgPath)
            if err != nil {
                return fmt.Errorf("create %s: %w", dbgPath, err)
            }
            w := bufio.NewWriter(f)
            _, _ = fmt.Fprintf(w, "# Planned commands (%d hosts x %d commands)\n", len(childHosts), len(mf.Commands))
            _, _ = fmt.Fprintln(w, "# Discovery: cat /etc/hosts")
            for _, host := range childHosts {
                if strings.TrimSpace(host) != "" { _, _ = fmt.Fprintf(w, "# child: %s\n", host) }
                for _, c := range mf.Commands {
                    shell := c.Shell
                    cmdArg := shellQuote(c.line())
                    if !strings.HasPrefix(cmdArg, "'") || !strings.HasSuffix(cmdArg, "'") {
                        cmdArg = "'" + cmdArg + "'"
                    }
                    full := fmt.Sprintf("sudo -u admin --shell %s -c %s", shellQuote(shell), cmdArg)
                    _ = full // for clarity
                    _, _ = fmt.Fprintln(w, full)
                }
            }
            _ = w.Flush()
            _ = f.Close()
            _, _ = fmt.Fprintf(os.Stderr, "Noop mode: wrote planned commands to %s\n", dbgPath)
            _, _ = fmt.Fprintf(os.Stderr, "Done. Output written to %s\n", cfgOutPath)
            return nil
        }

        // Execute commands for each child host
        for _, host := range childHosts {
            for i, c := range mf.Commands {
                title := fmt.Sprintf("[%d/%d] %s", i+1, len(mf.Commands), c.line())
                _, _ = fmt.Fprintf(os.Stderr, "Executing %s\n", title)

                // Write a combined heading for child host and optional title once per section
                if strings.TrimSpace(host) != "" || strings.TrimSpace(c.Title) != "" {
                    bw := bufio.NewWriter(outFile)
                    _, _ = fmt.Fprintln(bw, strings.Repeat("-", 80))
                    if strings.TrimSpace(host) != "" { _, _ = fmt.Fprintf(bw, "Child Host: %s\n", host) }
                    if strings.TrimSpace(c.Title) != "" { _, _ = fmt.Fprintf(bw, "Title: %s\n", c.Title) }
                    _ = bw.Flush()
                }

                shell := c.Shell
                cmdArg := shellQuote(c.line())
                if !strings.HasPrefix(cmdArg, "'") || !strings.HasSuffix(cmdArg, "'") {
                    cmdArg = "'" + cmdArg + "'"
                }
                fullCmd := fmt.Sprintf("sudo -u admin --shell %s -c %s", shellQuote(shell), cmdArg)

                cmdTimeout := c.perCommandTimeout(cfgTimeout)
                out, exitCode, runErr := runRemoteCommandFunc(sessClient, fullCmd, cmdTimeout)

                // Write section to output file
                if err := writeCommandSection(outFile, c, out, exitCode, runErr, cmdTimeout); err != nil {
                    return fmt.Errorf("failed writing output: %w", err)
                }

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

		_, _ = fmt.Fprintf(os.Stderr, "Done. Output written to %s\n", cfgOutPath)
		return nil
	},
}

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "arbor-exfil",
	Short:   "Run ArbOS commands from a YAML manifest and capture output",
	Long:    "Connects to an Arbor TMS leader (ArbOS) over SSH, executes commands from a YAML manifest, and writes responses to an output file.",
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
		defer func() {
			if client != nil {
				_ = client.Close()
			}
		}()

		// Execute commands
		for i, c := range mf.Commands {
			title := fmt.Sprintf("[%d/%d] %s", i+1, len(mf.Commands), c.line())
			_, _ = fmt.Fprintf(os.Stderr, "Executing %s\n", title)

			cmdTimeout := c.perCommandTimeout(cfgTimeout)
			out, exitCode, runErr := runRemoteCommandFunc(sshClientWrapper{client}, c.line(), cmdTimeout)

			// Write section to output file
			if err := writeCommandSection(outFile, c, out, exitCode, runErr, cmdTimeout); err != nil {
				return fmt.Errorf("failed writing output: %w", err)
			}

			// On timeout, try to reconnect once and continue
			if errors.Is(runErr, context.DeadlineExceeded) {
				_, _ = fmt.Fprintln(os.Stderr, "Command timed out; reconnecting...")
				if client != nil {
					_ = client.Close()
				}
				client, err = dialSSHFunc(cfgTarget, cfgUser, cfgPassword, cfgKeyPath, cfgPassphrase, cfgKnownHosts, cfgStrictHost, cfgConnTimeout)
				if err != nil {
					return fmt.Errorf("reconnect failed after timeout: %w", err)
				}
			}
		}

		_, _ = fmt.Fprintf(os.Stderr, "Done. Output written to %s\n", cfgOutPath)
		return nil
	},
}

package cmd

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// init configures the root command's persistent flags, binds them to environment
// variables via Viper, and registers all subcommands. This wiring ensures a
// consistent configuration surface across run/verify/install and keeps
// environment overrides predictable for operators.
func init() {
    // Persistent flags (inherited by subcommands like `run`)
    rootCmd.PersistentFlags().StringVarP(&cfgTarget, "target", "t", "", "ArbOS target FQDN/IP:port (e.g., tms.example.com:22)")
    rootCmd.PersistentFlags().StringVarP(&cfgManifest, "manifest", "m", "", "Path to YAML manifest file")
    rootCmd.PersistentFlags().StringVarP(&cfgOutPath, "out", "o", "", "Path to output text file")
    rootCmd.PersistentFlags().StringVarP(&cfgUser, "user", "u", "", "SSH username")
    rootCmd.PersistentFlags().StringVar(&cfgPassword, "password", "", "SSH password (or set ARBOR_EXFIL_PASSWORD)")
    rootCmd.PersistentFlags().StringVar(&cfgKeyPath, "key", "", "Path to SSH private key (PEM, OpenSSH)")
    rootCmd.PersistentFlags().StringVar(&cfgPassphrase, "passphrase", "", "Private key passphrase (or set ARBOR_EXFIL_PASSPHRASE)")
    rootCmd.PersistentFlags().StringVar(&cfgKnownHosts, "known-hosts", filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"), "Path to known_hosts file")
    rootCmd.PersistentFlags().BoolVar(&cfgStrictHost, "strict-host-key", true, "Require host key verification (disable to accept any host key)")
    rootCmd.PersistentFlags().DurationVar(&cfgTimeout, "cmd-timeout", 0, "Per-command timeout (e.g., 30s). 0 disables")
    rootCmd.PersistentFlags().DurationVar(&cfgConnTimeout, "conn-timeout", 15*time.Second, "Connection timeout")
    rootCmd.PersistentFlags().BoolVar(&cfgNoop, "noop", false, "Do not execute commands; write planned command strings to debug.out")
    rootCmd.PersistentFlags().StringVar(&cfgInstallPubKeyPath, "install-pubkey", "", "Path to SSH public key to install for arbor-exfil user (install subcommand)")

	// Bind env with Viper
    _ = viper.BindPFlag("target", rootCmd.PersistentFlags().Lookup("target"))
    _ = viper.BindPFlag("manifest", rootCmd.PersistentFlags().Lookup("manifest"))
    _ = viper.BindPFlag("out", rootCmd.PersistentFlags().Lookup("out"))
    _ = viper.BindPFlag("user", rootCmd.PersistentFlags().Lookup("user"))
    _ = viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
    _ = viper.BindPFlag("key", rootCmd.PersistentFlags().Lookup("key"))
    _ = viper.BindPFlag("passphrase", rootCmd.PersistentFlags().Lookup("passphrase"))
    _ = viper.BindPFlag("known-hosts", rootCmd.PersistentFlags().Lookup("known-hosts"))
    _ = viper.BindPFlag("strict-host-key", rootCmd.PersistentFlags().Lookup("strict-host-key"))
    _ = viper.BindPFlag("cmd-timeout", rootCmd.PersistentFlags().Lookup("cmd-timeout"))
    _ = viper.BindPFlag("conn-timeout", rootCmd.PersistentFlags().Lookup("conn-timeout"))
    _ = viper.BindPFlag("noop", rootCmd.PersistentFlags().Lookup("noop"))
    _ = viper.BindPFlag("install-pubkey", rootCmd.PersistentFlags().Lookup("install-pubkey"))

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
			if d, err := time.ParseDuration(v); err == nil {
				cfgTimeout = d
			}
		}
        if v := viper.GetString("conn-timeout"); v != "" {
            if d, err := time.ParseDuration(v); err == nil {
                cfgConnTimeout = d
            }
        }
        // Booleans
        if viper.IsSet("strict-host-key") {
            cfgStrictHost = viper.GetBool("strict-host-key")
        }
        if viper.IsSet("noop") {
            cfgNoop = viper.GetBool("noop")
        }
        if v := viper.GetString("install-pubkey"); v != "" {
            cfgInstallPubKeyPath = v
        }
    })

    // Add subcommands
    rootCmd.AddCommand(runCmd)
    rootCmd.AddCommand(verifyCmd)
    rootCmd.AddCommand(installCmd)
}

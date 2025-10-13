package cmd

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
	})
}

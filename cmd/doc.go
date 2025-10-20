// Package cmd implements the arbor-exfil command-line interface.
//
// The package organizes all CLI subcommands (run, verify, install) and the
// underlying helpers for SSH connectivity, persistent PTY shell execution,
// discovery of hosts via /etc/hosts, and structured YAML report emission.
//
// New contributors should start by reading rootCmd.go to see how cobra is
// wired, runCmd.go for the main execution flow, discovery.go for how child
// hosts are discovered, and persistentShell.go for the single-connection
// persistent interactive shell used to execute multiple commands reliably.
package cmd


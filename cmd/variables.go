package cmd

import (
	"errors"
	"time"
)

// Version is the CLI version string injected at build time via -ldflags.
var Version = "0.1.0"
// errAdminUser signals that privileged accounts must not be used for
// automation to encourage least-privilege operational practices.
var errAdminUser = errors.New("admin user account cannot be used")

var (
    // Global configuration populated by flags and/or environment variables.
    // These are declared here so they are visible across subcommands.
    cfgManifest    string
    cfgTarget      string
    cfgUser        string
    cfgPassword    string
    cfgKeyPath     string
    cfgPassphrase  string
    cfgOutPath     string
    cfgKnownHosts  string
    cfgStrictHost  bool
    cfgTimeout     time.Duration
    cfgConnTimeout time.Duration
    cfgNoop        bool
    cfgInstallPubKeyPath string
)

// Allow tests to stub dialing and command execution
var (
	dialSSHFunc          = dialSSH
	runRemoteCommandFunc = runRemoteCommand
)

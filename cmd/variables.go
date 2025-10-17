package cmd

import (
	"errors"
	"time"
)

// Version is set via -ldflags at build time if desired
var Version = "0.1.0"
var errAdminUser = errors.New("admin user account cannot be used")

var (
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
)

// Allow tests to stub dialing and command execution
var (
	dialSSHFunc          = dialSSH
	runRemoteCommandFunc = runRemoteCommand
)

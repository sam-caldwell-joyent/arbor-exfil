package cmd

import (
    "testing"

    "github.com/stretchr/testify/require"
)

// TestVerify_Succeeds_OnValidManifest verifies that the verify subcommand
// accepts a manifest with commands and required shell fields. Assumes local
// temp manifest and no network.
func TestVerify_Succeeds_OnValidManifest(t *testing.T) {
    resetConfig()
    tmp := t.TempDir()
    mf := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - title: T
    command: echo
    shell: /bin/sh
    args: ["ok"]
`)
    rootCmd.SetArgs([]string{"verify", "--manifest", mf})
    require.NoError(t, rootCmd.Execute())
}

// TestVerify_AllowsDiscoveryOnly_NoCommands verifies that a manifest with an
// empty commands list is valid for discovery-only operation.
func TestVerify_AllowsDiscoveryOnly_NoCommands(t *testing.T) {
    resetConfig()
    tmp := t.TempDir()
    mf := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands: []
`)
    rootCmd.SetArgs([]string{"verify", "--manifest", mf})
    require.NoError(t, rootCmd.Execute())
}

// TestVerify_Errors_OnMissingShell verifies that verify rejects manifests where
// any command is missing the required shell attribute.
func TestVerify_Errors_OnMissingShell(t *testing.T) {
    resetConfig()
    tmp := t.TempDir()
    mf := writeTemp(t, tmp, "m.yaml", `
name: N
description: D
commands:
  - command: echo
`)
    rootCmd.SetArgs([]string{"verify", "--manifest", mf})
    err := rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "commands[0].shell is required")
}

// TestVerify_Fails_OnMissingRequiredFields verifies that missing top-level
// required fields (e.g., description) cause verify to error.
func TestVerify_Fails_OnMissingRequiredFields(t *testing.T) {
    resetConfig()
    tmp := t.TempDir()
    mf := writeTemp(t, tmp, "m.yaml", `
name: N
commands:
  - command: x
    shell: /bin/sh
`)
    rootCmd.SetArgs([]string{"verify", "--manifest", mf})
    err := rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "manifest.description is required")
}

// TestVerify_RequiresManifestPath verifies that verify requires the --manifest
// flag/path and errors when omitted.
func TestVerify_RequiresManifestPath(t *testing.T) {
    resetConfig()
    rootCmd.SetArgs([]string{"verify"})
    err := rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "--manifest is required")
}

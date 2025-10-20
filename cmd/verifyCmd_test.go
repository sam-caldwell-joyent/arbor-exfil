package cmd

import (
    "testing"

    "github.com/stretchr/testify/require"
)

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

func TestVerify_RequiresManifestPath(t *testing.T) {
    resetConfig()
    rootCmd.SetArgs([]string{"verify"})
    err := rootCmd.Execute()
    require.Error(t, err)
    require.Contains(t, err.Error(), "--manifest is required")
}

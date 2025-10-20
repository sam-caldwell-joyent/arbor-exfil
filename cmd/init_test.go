package cmd

import (
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/require"
)

// TestInit_EnvOverrides_Dedicated verifies that environment variables are read
// during cobra/viper initialization and populate the corresponding config
// values. Assumes Execute() triggers OnInitialize without performing network.
func TestInit_EnvOverrides_Dedicated(t *testing.T) {
    resetConfig()
    t.Setenv("ARBOR_EXFIL_PASSWORD", "s3cr3t")
    t.Setenv("ARBOR_EXFIL_PASSPHRASE", "pp")

    tmp := t.TempDir()
    // Trigger OnInitialize by executing with minimal args that error after init
    rootCmd.SetArgs([]string{
        "--manifest", filepath.Join(tmp, "missing.yaml"),
        "--out", filepath.Join(tmp, "out.txt"),
    })
    _ = rootCmd.Execute()
    require.Equal(t, "s3cr3t", cfgPassword)
    require.Equal(t, "pp", cfgPassphrase)
}

// TestInit_EnvOverrides_AllFlags verifies that all supported environment
// variables and flags are bound into config, including timeouts and
// known-hosts via flag. Assumes viper default behavior for hyphenated keys.
func TestInit_EnvOverrides_AllFlags(t *testing.T) {
    resetConfig()
    t.Setenv("ARBOR_EXFIL_TARGET", "1.2.3.4:22")
    t.Setenv("ARBOR_EXFIL_MANIFEST", "/tmp/m.yaml")
    t.Setenv("ARBOR_EXFIL_OUT", "/tmp/out.txt")
    t.Setenv("ARBOR_EXFIL_USER", "u")
    t.Setenv("ARBOR_EXFIL_PASSWORD", "p")
    t.Setenv("ARBOR_EXFIL_KEY", "/k")
    t.Setenv("ARBOR_EXFIL_PASSPHRASE", "pp")

    // Force init path to run and set known-hosts via flag (env var with hyphen
    // key is not automatically mapped by viper without a custom replacer)
    rootCmd.SetArgs([]string{"--manifest", "/nope", "--out", "/nope", "--known-hosts", "/kh", "--cmd-timeout", "2s", "--conn-timeout", "1s"})
    _ = rootCmd.Execute()

    require.Equal(t, "1.2.3.4:22", cfgTarget)
    require.Equal(t, "/tmp/m.yaml", cfgManifest)
    require.Equal(t, "/tmp/out.txt", cfgOutPath)
    require.Equal(t, "u", cfgUser)
    require.Equal(t, "p", cfgPassword)
    require.Equal(t, "/k", cfgKeyPath)
    require.Equal(t, "pp", cfgPassphrase)
    require.Equal(t, "/kh", cfgKnownHosts)
    // strict-host-key remains default (true) unless set via flag; env var for hyphenated keys
    // is not automatically mapped by viper without a custom replacer.
    require.Equal(t, int64(2_000_000_000), int64(cfgTimeout)) // 2s in ns
    require.Equal(t, int64(1_000_000_000), int64(cfgConnTimeout)) // 1s in ns
}

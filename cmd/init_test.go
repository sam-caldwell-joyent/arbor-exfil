package cmd

import (
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/require"
)

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


package cmd

import (
    "errors"
    "fmt"
    "os"
    "strings"

    "github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
    Use:   "verify",
    Short: "Validate a manifest YAML file",
    RunE: func(cmd *cobra.Command, args []string) error {
        if cfgManifest == "" {
            return errors.New("--manifest is required (path to YAML)")
        }
        mf, err := loadManifest(cfgManifest)
        if err != nil {
            return fmt.Errorf("invalid manifest: %w", err)
        }
        // Additional validation aligned with runtime behavior: each command must specify a shell.
        if len(mf.Commands) > 0 {
            for i, c := range mf.Commands {
                if strings.TrimSpace(c.Shell) == "" {
                    return fmt.Errorf("invalid manifest: commands[%d].shell is required", i)
                }
            }
        }
        _, _ = fmt.Fprintln(os.Stdout, "Manifest OK")
        return nil
    },
}


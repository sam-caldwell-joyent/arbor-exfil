package cmd

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "arbor-exfil",
    Short: "Run ArbOS commands from a YAML manifest and capture output",
    Long: "Connects to an Arbor TMS leader (ArbOS) over SSH, executes commands from a YAML manifest, and writes " +
        "responses to an output file.",
    Version: Version,
    Run: func(cmd *cobra.Command, args []string) {
        // Show help by default and hint to use subcommands
        _ = cmd.Help()
    },
}

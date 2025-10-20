package cmd

import (
	"errors"
	"fmt"
	"os"
)

func Execute() {
    // Execute runs the cobra root command tree (arbor-exfil CLI). It captures
    // errors and maps them to process exit codes while providing user-friendly
    // messages. Admin user errors are written to stdout to make the constraint
    // explicit to operators; all other errors go to stderr.
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, errAdminUser) {
			// Admin account error prints to stdout and exits with code 1
			_, _ = fmt.Fprintln(os.Stdout, err.Error())
			exitFunc(1)
			return
		}
		_, _ = fmt.Fprintln(os.Stderr, err)
		exitFunc(1)
		return
	}
}

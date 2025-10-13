package cmd

import (
	"errors"
	"fmt"
	"os"
)

func Execute() {
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

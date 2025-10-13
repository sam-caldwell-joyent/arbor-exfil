package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

// writeCommandSection writes one command's results
func writeCommandSection(w io.Writer, c commandEntry, out []byte, exitCode int, runErr error, timeout time.Duration) error {
	bw := bufio.NewWriter(w)
	_, _ = fmt.Fprintln(bw, strings.Repeat("-", 80))
	_, _ = fmt.Fprintf(bw, "Command: %s\n", c.line())
	if timeout > 0 {
		_, _ = fmt.Fprintf(bw, "Timeout: %s\n", timeout.String())
	}
	_, _ = fmt.Fprintf(bw, "Exit Code: %d\n", exitCode)
	if runErr != nil {
		_, _ = fmt.Fprintf(bw, "Error: %v\n", runErr)
	}
	_, _ = fmt.Fprintln(bw, "Output:")
	_, _ = fmt.Fprintln(bw, "---8<---")
	// Write raw output
	_, _ = bw.Write(out)
	if len(out) == 0 || !strings.HasSuffix(string(out), "\n") {
		_, _ = bw.WriteString("\n")
	}
	_, _ = fmt.Fprintln(bw, "---8<---")
	return bw.Flush()
}

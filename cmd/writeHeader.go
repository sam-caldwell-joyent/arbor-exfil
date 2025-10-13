package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

// writeHeader writes manifest meta info to the output file
func writeHeader(w io.Writer, mf *manifest) {
	bw := bufio.NewWriter(w)
	_, _ = fmt.Fprintf(bw, "Name: %s\n", mf.Name)
	_, _ = fmt.Fprintf(bw, "Description: %s\n", mf.Description)
	_, _ = fmt.Fprintf(bw, "Generated: %s\n", time.Now().Format(time.RFC3339))
	_, _ = fmt.Fprintf(bw, "Command Count: %d\n", len(mf.Commands))
	_, _ = fmt.Fprintln(bw, strings.Repeat("=", 80))
	_ = bw.Flush()
}

package cmd

import (
    "bytes"
    "strings"
    "testing"
)

func TestWriteCommandSection_WithTitle_NoDuplicateSeparator(t *testing.T) {
    var buf bytes.Buffer
    // Simulate pre-run heading write (as rootCmd does)
    buf.WriteString(strings.Repeat("-", 80) + "\n")
    buf.WriteString("Title: My Task\n")
    // Now write the command section; separator should NOT be duplicated
    if err := writeCommandSection(&buf, commandEntry{Command: "echo 1", Title: "My Task"}, []byte("ok\n"), 0, nil, 0); err != nil {
        t.Fatalf("writeCommandSection error: %v", err)
    }
    out := buf.String()
    if strings.Count(out, strings.Repeat("-", 80)) != 1 {
        t.Fatalf("expected single separator, got %d", strings.Count(out, strings.Repeat("-", 80)))
    }
    if !strings.Contains(out, "Title: My Task\n") {
        t.Fatalf("missing title heading")
    }
    if !strings.Contains(out, "Command: echo 1\n") {
        t.Fatalf("missing command line")
    }
}


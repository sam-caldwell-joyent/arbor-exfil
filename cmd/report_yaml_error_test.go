package cmd

import (
    "errors"
    "testing"
)

type failingWriter struct{}

func (f failingWriter) Write(p []byte) (int, error) { return 0, errors.New("write fail") }

// TestWriteYAMLReport_ErrorOnWrite verifies that writeYAMLReport propagates an
// error from the underlying writer when encoding the YAML document. Assumes a
// minimal report and a writer that always fails.
func TestWriteYAMLReport_ErrorOnWrite(t *testing.T) {
    rep := newYAMLReport(&manifest{Name: "N", Description: "D"})
    if err := writeYAMLReport(failingWriter{}, rep); err == nil {
        t.Fatalf("expected write error, got nil")
    }
}

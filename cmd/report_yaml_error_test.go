package cmd

import (
    "errors"
    "testing"
)

type failingWriter struct{}

func (f failingWriter) Write(p []byte) (int, error) { return 0, errors.New("write fail") }

func TestWriteYAMLReport_ErrorOnWrite(t *testing.T) {
    rep := newYAMLReport(&manifest{Name: "N", Description: "D"})
    if err := writeYAMLReport(failingWriter{}, rep); err == nil {
        t.Fatalf("expected write error, got nil")
    }
}

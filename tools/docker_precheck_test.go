package tools

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "testing"
    "time"
)

// TestMain provides a fast, centralized Docker availability check for the tools
// test suite. If Docker is not installed or the daemon is unreachable, fail
// immediately so dependent tests do not hang or report late failures.
func TestMain(m *testing.M) {
    // Look for docker binary
    if _, err := exec.LookPath("docker"); err != nil {
        fmt.Fprintln(os.Stderr, "Docker not found in PATH; failing fast")
        os.Exit(1)
    }

    // Verify daemon is reachable quickly
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := exec.CommandContext(ctx, "docker", "info").Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Docker daemon not available: %v\n", err)
        os.Exit(1)
    }

    os.Exit(m.Run())
}

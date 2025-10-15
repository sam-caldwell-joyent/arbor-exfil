package tools

import (
    "context"
    "os/exec"
    "strings"
    "testing"
    "time"
)

// TestDockerBuild_LocalImage builds the container image using the Dockerfile
// at repo root and ensures it is tagged as arbor-exfil:local-test. The test is
// skipped if docker is not available or when running with -short.
func TestDockerBuild_LocalImage(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping docker build in -short mode")
    }
    if _, err := exec.LookPath("docker"); err != nil {
        t.Skip("docker not found in PATH; skipping container build test")
    }

    // Ensure Docker daemon is reachable; if not, skip instead of failing CI.
    probeCtx, probeCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer probeCancel()
    probe := exec.CommandContext(probeCtx, "docker", "info")
    if err := probe.Run(); err != nil {
        t.Skipf("docker daemon not available: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()

    // Build image
    build := exec.CommandContext(ctx, "docker", "build", "-t", "arbor-exfil:local-test", ".")
    out, err := build.CombinedOutput()
    if ctx.Err() == context.DeadlineExceeded {
        t.Fatalf("docker build timed out")
    }
    if err != nil {
        t.Fatalf("docker build failed: %v\n%s", err, string(out))
    }

    // Verify image exists
    inspect := exec.CommandContext(ctx, "docker", "image", "inspect", "arbor-exfil:local-test", "--format={{.Id}}")
    out2, err := inspect.CombinedOutput()
    if ctx.Err() == context.DeadlineExceeded {
        t.Fatalf("docker image inspect timed out")
    }
    if err != nil {
        t.Fatalf("docker image inspect failed: %v\n%s", err, string(out2))
    }
    if strings.TrimSpace(string(out2)) == "" {
        t.Fatalf("docker image inspect returned empty id")
    }
}


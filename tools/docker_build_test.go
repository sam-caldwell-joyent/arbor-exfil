package tools

import (
	"context"
	"errors"
	"os"
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

	// Ensure required base images are present or pullable; otherwise skip to
	// avoid false negatives in locked-down CI environments (custom CAs, etc.)
	ensureImage := func(img string) {
		// If image exists locally, return
		ic, icCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer icCancel()
		if err := exec.CommandContext(ic, "docker", "image", "inspect", img, "--format={{.Id}}").Run(); err == nil {
			return
		}
		// Attempt to pull; skip on failure
		pc, pcCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer pcCancel()
		pull := exec.CommandContext(pc, "docker", "pull", img)
		if out, err := pull.CombinedOutput(); err != nil {
			t.Skipf("skipping: cannot pull base image %s: %v\n%s", img, err, string(out))
		}
	}
	ensureImage("ubuntu:22.04")
	ensureImage("gcr.io/distroless/static:nonroot")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Build image
	// Use repo root (parent of tools/) as build context so Dockerfile is found
	build := exec.CommandContext(ctx, "docker", "build", "-t", "arbor-exfil:local-test", "..")
	build.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	out, err := build.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("docker build timed out")
	}
	if err != nil {
		so := string(out)
		if strings.Contains(so, "certificate signed by unknown authority") ||
			strings.Contains(so, "x509:") ||
			strings.Contains(so, "TLS handshake timeout") ||
			strings.Contains(so, "connection refused") ||
			strings.Contains(so, "no route to host") {
			t.Skipf("skipping: docker build environment issue: %v\n%s", err, so)
		}
		t.Fatalf("docker build failed: %v\n%s", err, so)
	}

	// Verify image exists
	inspect := exec.CommandContext(ctx, "docker", "image", "inspect", "arbor-exfil:local-test", "--format={{.Id}}")
	out2, err := inspect.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("docker image inspect timed out")
	}
	if err != nil {
		t.Fatalf("docker image inspect failed: %v\n%s", err, string(out2))
	}
	if strings.TrimSpace(string(out2)) == "" {
		t.Fatalf("docker image inspect returned empty id")
	}
}

package cmd

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "gopkg.in/yaml.v3"
    srv "arbor-exfil/tools/sshserv"
)

// TestEndToEnd_WithLocalTestServer starts the test SSH server on 127.0.0.1:20222,
// runs the CLI against the provided inspection_report.yaml, and verifies that
// each command yields "ok" and exit code 0, and that the overall output file
// is created with the expected sections.
func TestEndToEnd_WithLocalTestServer(t *testing.T) {
    // Start local test ssh server
    stop, err := srv.Start("127.0.0.1:20222")
    if err != nil {
        t.Skipf("skipping e2e: cannot start test ssh server: %v", err)
    }
    defer stop()
    // Give it a moment to bind
    time.Sleep(100 * time.Millisecond)

    resetConfig()

    // Prepare output file path and an ad-hoc manifest with titles
    tmp := t.TempDir()
    outPath := filepath.Join(tmp, "out.txt")
    mfPath := writeTemp(t, tmp, "inspection.yaml", `
name: E2E Test
description: Test commands
commands:
  - title: First
    command: cmd1
    shell: /bin/sh
  - title: Second
    command: cmd2
    shell: /bin/sh
  - title: Third
    command: cmd3
    shell: /bin/sh
  - title: Fourth
    command: cmd4
    shell: /bin/sh
  - title: Fifth
    command: cmd5
    shell: /bin/sh
`)

    // Set CLI args
    rootCmd.SetArgs([]string{
        "--target", "127.0.0.1:20222",
        "--user", "tester",
        "--manifest", mfPath,
        "--out", outPath,
        "--strict-host-key=false",
    })

    if err := rootCmd.Execute(); err != nil {
        t.Fatalf("rootCmd execute failed: %v", err)
    }

    b, err := os.ReadFile(outPath)
    if err != nil { t.Fatalf("read out: %v", err) }
    var rep yamlReport
    if err := yaml.Unmarshal(b, &rep); err != nil {
        t.Fatalf("unmarshal yaml: %v\n%s", err, string(b))
    }
    // We expect five successful results overall
    success := 0
    for _, run := range rep.Runs {
        for _, r := range run.Results {
            if r.ExitCode == 0 && (r.Output == "ok\n" || r.Output == "ok\r\n") {
                success++
            }
        }
    }
    if success < 5 { t.Fatalf("expected at least 5 ok results, got %d", success) }
}

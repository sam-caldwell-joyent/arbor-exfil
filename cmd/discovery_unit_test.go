package cmd

import (
    "errors"
    "testing"
    "time"
)

// stub client that we never use directly (runRemoteCommandFunc is stubbed)
type noopClient struct{}
func (noopClient) NewSession() (session, error) { return nil, nil }

// TestDiscoverChildHosts_SuccessAndError verifies that host discovery returns
// raw hosts content, parses and filters IPs (dropping ::1), and propagates
// errors while preserving raw output and exit code. Assumes runRemoteCommandFunc
// is stubbed and no real SSH connection occurs.
func TestDiscoverChildHosts_SuccessAndError(t *testing.T) {
    origRun := runRemoteCommandFunc
    t.Cleanup(func() { runRemoteCommandFunc = origRun })

    // Success path
    calls := 0
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        calls++
        return []byte("127.0.0.1 a\n10.0.0.2 b\n::1 c\n"), 0, nil
    }
    raw, code, ips, err := discoverChildHosts(noopClient{}, 0)
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if code != 0 { t.Fatalf("want exit 0, got %d", code) }
    if len(raw) == 0 { t.Fatalf("want raw hosts") }
    if len(ips) != 2 { t.Fatalf("want 2 ips (filter ::1), got %d", len(ips)) }

    // Error path
    runRemoteCommandFunc = func(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
        return []byte("oops"), 7, errors.New("bad")
    }
    raw, code, ips, err = discoverChildHosts(noopClient{}, 0)
    if err == nil { t.Fatalf("expected error") }
    if code != 7 { t.Fatalf("want exit 7, got %d", code) }
    if string(raw) != "oops" { t.Fatalf("preserve raw on error") }
    if ips != nil { t.Fatalf("ips should be nil on error") }
}

package cmd

import (
    "context"
    "errors"
    "time"

	"golang.org/x/crypto/ssh"
)

// runRemoteCommand executes cmd on a new session obtained from client and
// returns the combined output, exit code, and error. When timeout is > 0, the
// operation is bounded and returns context.DeadlineExceeded on timeout.
//
// For persistent sessions (backed by a PTY shell) we prefer an explicit exit
// code via the optional LastExitCode interface; otherwise we derive it from
// ssh.ExitError when available.
func runRemoteCommand(client sessionClient, cmd string, timeout time.Duration) ([]byte, int, error) {
	type result struct {
		out      []byte
		exitCode int
		err      error
	}

	run := func() result {
		currSession, err := client.NewSession()
		if err != nil {
			return result{nil, -1, err}
		}
    // Best-effort close; ignore errors to avoid panics under EOF/exit-status-only flows
    defer func(thisSession session) { _ = thisSession.Close() }(currSession)
        b, err := currSession.CombinedOutput(cmd)
        // Prefer exit code via optional interface (persistent sessions)
        if ec, ok := currSession.(interface{ LastExitCode() int }); ok {
            return result{b, ec.LastExitCode(), err}
        }
        if err == nil {
            return result{b, 0, nil}
        }
        // Try to derive exit status from ssh.ExitError
        exit := -1
        var ee *ssh.ExitError
        if errors.As(err, &ee) {
            exit = ee.ExitStatus()
        }
        return result{b, exit, err}
	}

	if timeout <= 0 {
		r := run()
		return r.out, r.exitCode, r.err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ch := make(chan result, 1)
	go func() { ch <- run() }()

	select {
	case r := <-ch:
		return r.out, r.exitCode, r.err
	case <-ctx.Done():
		// Best-effort: indicate timeout. Caller may reconnect if desired.
		return nil, -1, context.DeadlineExceeded
	}
}

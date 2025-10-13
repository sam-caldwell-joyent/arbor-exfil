package cmd

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/ssh"
)

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
		defer func(thisSession session) {
			err := thisSession.Close()
			if err != nil {
				panic(err)
			}
		}(currSession)
		b, err := currSession.CombinedOutput(cmd)
		if err == nil {
			return result{b, 0, nil}
		}
		// Try to derive exit status
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

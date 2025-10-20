package cmd

import (
    "context"
    "errors"
    "testing"
    "time"
    "github.com/stretchr/testify/require"
)

// TestRunRemoteCommand_Success_Dedicated verifies that runRemoteCommand returns
// stdout, exit code 0, and closes the session on success. Assumes a stubbed
// session/client and no real SSH network activity.
func TestRunRemoteCommand_Success_Dedicated(t *testing.T) {
    s := &fakeSession{out: []byte("OK\n"), err: nil}
    out, code, err := runRemoteCommand(&fakeClient{sess: s}, "echo OK", 0)
    require.NoError(t, err)
    require.Equal(t, 0, code)
    require.Equal(t, "OK\n", string(out))
    require.True(t, s.closed)
}

// TestRunRemoteCommand_Timeout_Dedicated verifies that a command exceeding the
// provided timeout returns an error (DeadlineExceeded), an exit code of -1, and
// no output. Assumes a stubbed session that delays its response.
func TestRunRemoteCommand_Timeout_Dedicated(t *testing.T) {
    s := &fakeSession{out: []byte("SLOW\n"), delay: 200 * time.Millisecond}
    out, code, err := runRemoteCommand(&fakeClient{sess: s}, "sleep", 10*time.Millisecond)
    require.Error(t, err)
    require.True(t, errors.Is(err, context.DeadlineExceeded))
    require.Equal(t, -1, code)
    require.Nil(t, out)
}

// TestRunRemoteCommand_NewSessionError_Dedicated verifies that a failure to
// acquire a new SSH session returns an error and -1 exit code. Assumes no
// network and a client stub configured to fail on NewSession.
func TestRunRemoteCommand_NewSessionError_Dedicated(t *testing.T) {
    out, code, err := runRemoteCommand(&fakeClient{newErr: errors.New("no session")}, "cmd", 0)
    require.Error(t, err)
    require.Equal(t, -1, code)
    require.Nil(t, out)
}

// TestRunRemoteCommand_CommandError_NoExitCode_Dedicated verifies that when the
// command returns an error without a concrete exit status, the function returns
// -1 while preserving any stdout content. Assumes stubbed session/client.
func TestRunRemoteCommand_CommandError_NoExitCode_Dedicated(t *testing.T) {
    s := &fakeSession{out: []byte("oops\n"), err: errors.New("boom")}
    out, code, err := runRemoteCommand(&fakeClient{sess: s}, "cmd", 0)
    require.Error(t, err)
    require.Equal(t, -1, code)
    require.Equal(t, "oops\n", string(out))
}

// A session that exposes LastExitCode should have its code preferred even when
// CombinedOutput returns no error (used by persistent PTY session path).
type _lastExitSession struct{ out []byte; code int }

func (s *_lastExitSession) CombinedOutput(cmd string) ([]byte, error) { return s.out, nil }
func (s *_lastExitSession) Close() error { return nil }
func (s *_lastExitSession) LastExitCode() int { return s.code }

type _lastExitClient struct{ sess *_lastExitSession }

func (c *_lastExitClient) NewSession() (session, error) { return c.sess, nil }

// TestRunRemoteCommand_PrefersLastExitCode verifies that when the session
// exposes a LastExitCode, that value is preferred for the exit status even if
// CombinedOutput itself does not return an error. Assumes a stubbed session.
func TestRunRemoteCommand_PrefersLastExitCode(t *testing.T) {
    s := &_lastExitSession{out: []byte("Z\n"), code: 42}
    out, code, err := runRemoteCommand(&_lastExitClient{sess: s}, "cmd", 0)
    require.NoError(t, err)
    require.Equal(t, 42, code)
    require.Equal(t, "Z\n", string(out))
}

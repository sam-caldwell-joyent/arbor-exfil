package cmd

import (
    "context"
    "errors"
    "testing"
    "time"
    "github.com/stretchr/testify/require"
)

func TestRunRemoteCommand_Success_Dedicated(t *testing.T) {
    s := &fakeSession{out: []byte("OK\n"), err: nil}
    out, code, err := runRemoteCommand(&fakeClient{sess: s}, "echo OK", 0)
    require.NoError(t, err)
    require.Equal(t, 0, code)
    require.Equal(t, "OK\n", string(out))
    require.True(t, s.closed)
}

func TestRunRemoteCommand_Timeout_Dedicated(t *testing.T) {
    s := &fakeSession{out: []byte("SLOW\n"), delay: 200 * time.Millisecond}
    out, code, err := runRemoteCommand(&fakeClient{sess: s}, "sleep", 10*time.Millisecond)
    require.Error(t, err)
    require.True(t, errors.Is(err, context.DeadlineExceeded))
    require.Equal(t, -1, code)
    require.Nil(t, out)
}

func TestRunRemoteCommand_NewSessionError_Dedicated(t *testing.T) {
    out, code, err := runRemoteCommand(&fakeClient{newErr: errors.New("no session")}, "cmd", 0)
    require.Error(t, err)
    require.Equal(t, -1, code)
    require.Nil(t, out)
}

func TestRunRemoteCommand_CommandError_NoExitCode_Dedicated(t *testing.T) {
    s := &fakeSession{out: []byte("oops\n"), err: errors.New("boom")}
    out, code, err := runRemoteCommand(&fakeClient{sess: s}, "cmd", 0)
    require.Error(t, err)
    require.Equal(t, -1, code)
    require.Equal(t, "oops\n", string(out))
}


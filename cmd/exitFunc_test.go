package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestExitFunc_OverrideAndCall(t *testing.T) {
    orig := exitFunc
    t.Cleanup(func() { exitFunc = orig })
    called := 0
    exitFunc = func(code int) { called = code }
    exitFunc(3)
    require.Equal(t, 3, called)
}


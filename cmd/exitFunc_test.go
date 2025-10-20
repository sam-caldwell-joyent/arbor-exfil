package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

// TestExitFunc_OverrideAndCall verifies that the overridable exit function can
// be swapped in tests and that invoking it records the provided exit code.
// Assumes exitFunc is a package-level variable used by Execute().
func TestExitFunc_OverrideAndCall(t *testing.T) {
    orig := exitFunc
    t.Cleanup(func() { exitFunc = orig })
    called := 0
    exitFunc = func(code int) { called = code }
    exitFunc(3)
    require.Equal(t, 3, called)
}

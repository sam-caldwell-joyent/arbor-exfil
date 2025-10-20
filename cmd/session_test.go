package cmd

import "testing"

// Compile-time check: fakeSession implements session
var _ session = (*fakeSession)(nil)

// TestSession_Interface_Exists verifies that the session interface is
// implemented by the fakeSession used in tests, guarding interface stability.
// Assumes no behavior is exercised.
func TestSession_Interface_Exists(t *testing.T) {
    // This test exists to ensure the interface remains in place.
}

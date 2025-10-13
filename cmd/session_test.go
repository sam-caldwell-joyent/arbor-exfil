package cmd

import "testing"

// Compile-time check: fakeSession implements session
var _ session = (*fakeSession)(nil)

func TestSession_Interface_Exists(t *testing.T) {
    // This test exists to ensure the interface remains in place.
}


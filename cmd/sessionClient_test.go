package cmd

import "testing"

// Compile-time check: fakeClient implements sessionClient
var _ sessionClient = (*fakeClient)(nil)

func TestSessionClient_Interface_Exists(t *testing.T) {
    // This test exists to ensure the interface remains in place.
}


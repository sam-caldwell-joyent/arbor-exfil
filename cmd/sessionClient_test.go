package cmd

import "testing"

// Compile-time check: fakeClient implements sessionClient
var _ sessionClient = (*fakeClient)(nil)

// TestSessionClient_Interface_Exists verifies that the sessionClient interface
// is satisfied by the fakeClient used in tests, ensuring the interface remains
// stable for stubbing. Assumes no runtime behavior is invoked.
func TestSessionClient_Interface_Exists(t *testing.T) {
    // This test exists to ensure the interface remains in place.
}

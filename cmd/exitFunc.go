package cmd

import "os"

// exitFunc allows tests to stub process exit behavior.
// Production code leaves this pointing at os.Exit, while tests replace it in
// order to capture exit codes without actually terminating the process.
var exitFunc = os.Exit

package cmd

import "os"

// exitFunc allows tests to stub process exit behavior
var exitFunc = os.Exit

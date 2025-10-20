package cmd

// session is a minimal interface for running a command and closing. It is
// satisfied by both real *ssh.Session instances and the persistent session
// adapters used in tests and the persistentShell implementation.
type session interface {
	CombinedOutput(cmd string) ([]byte, error)
	Close() error
}

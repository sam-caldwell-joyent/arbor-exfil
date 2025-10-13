package cmd

// session is a minimal interface for running a command and closing
type session interface {
	CombinedOutput(cmd string) ([]byte, error)
	Close() error
}

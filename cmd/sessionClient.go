package cmd

// runRemoteCommand executes a single command and returns output + exit code
// sessionClient is a minimal interface to obtain a command session
type sessionClient interface {
	NewSession() (session, error)
}

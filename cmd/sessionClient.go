package cmd

// sessionClient is a minimal interface to obtain a command session. It is
// implemented by wrappers around *ssh.Client and by persistent session
// adapters so the execution path can remain decoupled from the transport.
type sessionClient interface {
	NewSession() (session, error)
}

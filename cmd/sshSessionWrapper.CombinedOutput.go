package cmd

// CombinedOutput return session combined output to caller
func (w sshSessionWrapper) CombinedOutput(cmd string) ([]byte, error) {
	return w.s.CombinedOutput(cmd)
}

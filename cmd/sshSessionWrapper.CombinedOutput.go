package cmd

// CombinedOutput executes cmd on the underlying ssh.Session and returns the
// combined stdout and stderr output, mirroring the standard library's
// semantics for exec.Cmd.
func (w sshSessionWrapper) CombinedOutput(cmd string) ([]byte, error) {
	return w.s.CombinedOutput(cmd)
}

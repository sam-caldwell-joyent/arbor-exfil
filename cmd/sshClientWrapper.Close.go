package cmd

// Close closes the underlying ssh.Session associated with this wrapper.
func (w sshSessionWrapper) Close() error {
	return w.s.Close()
}

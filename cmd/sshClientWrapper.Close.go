package cmd

// Close - close session
func (w sshSessionWrapper) Close() error {
	return w.s.Close()
}

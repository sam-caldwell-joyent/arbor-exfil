package cmd

import "time"

// perCommandTimeout returns the timeout to use for this command, falling back
// to the provided default when the manifest does not specify or specifies an
// invalid duration string.
func (c *commandEntry) perCommandTimeout(defaultTimeout time.Duration) time.Duration {
	if c.Timeout == "" {
		return defaultTimeout
	}
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return defaultTimeout
	}
	return d
}

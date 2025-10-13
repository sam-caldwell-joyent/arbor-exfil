package cmd

import "time"

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

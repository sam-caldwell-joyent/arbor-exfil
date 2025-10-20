package cmd

import "strings"

// line builds the fully rendered command line by appending arguments with
// safe shell quoting. It does not include sudo/shell wrappers – callers are
// responsible for wrapping according to their execution context.
func (c *commandEntry) line() string {
	if len(c.Args) == 0 {
		return c.Command
	}
	// Quote args to be safe for remote shell
	quoted := make([]string, 0, len(c.Args))
	for _, a := range c.Args {
		quoted = append(quoted, shellQuote(a))
	}
	return strings.TrimSpace(c.Command + " " + strings.Join(quoted, " "))
}

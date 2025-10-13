package cmd

import "strings"

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

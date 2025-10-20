package cmd

import "strings"

// shellQuote minimally quotes an argument for POSIX shells. It leaves common
// safe characters unquoted and uses single-quoting with the standard `'\''`
// escape for embedded single quotes. The result is suitable for building
// robust remote command lines.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.IndexFunc(s, func(r rune) bool {
		// Safe chars: alnum, - _ . / @ : and commas
		if r >= 'a' && r <= 'z' {
			return false
		}
		if r >= 'A' && r <= 'Z' {
			return false
		}
		if r >= '0' && r <= '9' {
			return false
		}
		switch r {
		case '-', '_', '.', '/', '@', ':', ',', '+', '=':
			return false
		}
		return true
	}) == -1 {
		return s
	}
	// Single-quote, escaping embedded single quotes: ' -> '\''
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

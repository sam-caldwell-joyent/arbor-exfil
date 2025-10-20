package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// persistentShell maintains a single long-lived remote shell over one ssh.Session
// and executes commands sequentially, capturing combined output and exit status
// using a unique delimiter echoed after each command.
// persistentShell manages a single interactive PTY shell on the remote host
// and provides a runOne method to execute sequential commands on that same
// connection. This preserves state (cwd, env) across commands and avoids the
// overhead and incompatibilities of starting a new exec channel for each.
type persistentShell struct {
	sess   *ssh.Session
	stdin  io.WriteCloser
	pr     *io.PipeReader
	pw     *io.PipeWriter
	reader *bufio.Reader
	mu     sync.Mutex

	nonce string
	seq   int
}

// newPersistentShell creates and initializes a remote PTY shell attached to a
// single SSH session. It wires stdout/stderr into a unified stream and starts
// /bin/sh in script mode. The returned shell is ready to run commands.
func newPersistentShell(client *ssh.Client) (*persistentShell, error) {
	s, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	// Create a single combined stream for stdout+stderr
	pr, pw := io.Pipe()
	s.Stdout = pw
	s.Stderr = pw

	stdin, err := s.StdinPipe()
	if err != nil {
		_ = pw.Close()
		_ = s.Close()
		return nil, err
	}

	// Request a PTY so that commands that require a TTY behave correctly.
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echo to avoid command-echo noise
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := s.RequestPty("xterm", 80, 40, modes); err != nil {
		_ = stdin.Close()
		_ = pw.Close()
		_ = s.Close()
		return nil, err
	}

	// Start a shell that reads commands from stdin (script mode). PTY is active.
	if err := s.Start("/bin/sh -s -"); err != nil {
		_ = stdin.Close()
		_ = pw.Close()
		_ = s.Close()
		return nil, err
	}

	ps := &persistentShell{
		sess:   s,
		stdin:  stdin,
		pr:     pr,
		pw:     pw,
		reader: bufio.NewReader(pr),
		nonce:  makeNonce(),
	}
	return ps, nil
}

// Close attempts to gracefully terminate the remote shell and release
// resources. It is safe to call multiple times; errors from stream closures are
// ignored and the underlying ssh.Session Close result is returned.
func (ps *persistentShell) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// Best effort to terminate shell
	_, _ = io.WriteString(ps.stdin, "exit\n")
	_ = ps.stdin.Close()
	_ = ps.pw.Close()
	return ps.sess.Close()
}

// runOne executes a single command line in the persistent shell and returns
// its combined output and exit code.
// runOne executes a single line in the persistent shell and returns combined
// output and the command's exit code. A unique marker is echoed after the
// command to delimit out-of-band exit status reliably across PTY streams.
func (ps *persistentShell) runOne(line string) ([]byte, int, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	id := ps.seq
	ps.seq++
	marker := fmt.Sprintf("__ARBOR_END__%s__%d__", ps.nonce, id)

	// Send the command followed by a marker with the exit status
	// Use echo to print marker and $?; avoid printf portability issues.
	// Ensure each logical command ends with a newline for the shell to execute.
	cmd := fmt.Sprintf("%s; echo %s $?\n", line, shellQuote(marker))
	if _, err := io.WriteString(ps.stdin, cmd); err != nil {
		return nil, -1, err
	}

	// Read until we encounter the marker anywhere in the stream; everything before it is output.
	var out bytes.Buffer
	var accum strings.Builder
	for {
		chunk, err := ps.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Unexpected end of stream before marker
				return out.Bytes(), -1, io.EOF
			}
			return out.Bytes(), -1, err
		}
		accum.WriteString(chunk)
		a := accum.String()
		idx := strings.Index(a, marker+" ")
		if idx < 0 {
			// No marker yet; flush all but keep buffer from growing unbounded.
			// Keep only last len(marker)+10 to handle split markers.
			if accum.Len() > 8192 {
				s := accum.String()
				// flush everything except tail to output
				tailStart := len(s) - (len(marker) + 16)
				if tailStart < 0 {
					tailStart = 0
				}
				_, _ = out.WriteString(s[:tailStart])
				accum.Reset()
				accum.WriteString(s[tailStart:])
			}
			continue
		}
		// Output up to marker
		_, _ = out.WriteString(a[:idx])
		// Parse exit code after marker
		rest := a[idx+len(marker)+1:] // skip marker and following space
		// Extract up to newline
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			restLine := strings.TrimSpace(rest[:nl])
			exit := -1
			if v, perr := parseInt(restLine); perr == nil {
				exit = v
			}
			return out.Bytes(), exit, nil
		}
		// Should not happen with ReadString('\n'), but handle defensively
		exit := -1
		if v, perr := parseInt(strings.TrimSpace(rest)); perr == nil {
			exit = v
		}
		return out.Bytes(), exit, nil
	}
}

// parseInt converts a decimal string to int, safe for small values
// parseInt converts a decimal string to int. It is intentionally minimal to
// avoid pulling in strconv for this simple parser in the hot path.
func parseInt(s string) (int, error) {
	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a number: %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

// makeNonce returns a short, pseudo-random identifier used to create unique
// end markers in the combined PTY stream so we can reliably detect command
// completion and capture the exit code.
func makeNonce() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// persistentSessionClient returns virtual sessions that run on the same persistent shell.
type persistentSessionClient struct{ ps *persistentShell }

// NewSession returns a lightweight virtual session backed by the same
// persistent shell. Each virtual session maps CombinedOutput to runOne.
func (c persistentSessionClient) NewSession() (session, error) {
	return &persistentVirtualSession{ps: c.ps}, nil
}

// persistentVirtualSession adapts persistentShell to the session interface.
// persistentVirtualSession adapts a persistentShell to the session interface by
// implementing CombinedOutput and tracking the last exit code.
type persistentVirtualSession struct {
	ps       *persistentShell
	lastExit int
}

// CombinedOutput runs cmd via the persistent shell and records the exit code
// for later retrieval by runRemoteCommand.
func (s *persistentVirtualSession) CombinedOutput(cmd string) ([]byte, error) {
	out, code, err := s.ps.runOne(cmd)
	s.lastExit = code
	return out, err
}

// Close is a no-op for the virtual session; the underlying persistent shell is
// owned by the session client and closed separately.
func (s *persistentVirtualSession) Close() error { return nil }

// LastExitCode exposes the exit status of the last command for runRemoteCommand.
// LastExitCode reports the last command's exit code as observed in the PTY
// stream marker.
func (s *persistentVirtualSession) LastExitCode() int { return s.lastExit }

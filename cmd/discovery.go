package cmd

import (
    "bufio"
    "bytes"
    "net"
    "strings"
    "time"
)

// parseHostsIPs returns a deduplicated, in-order list of IPs from /etc/hosts content.
// It parses the first non-comment, non-empty token on each line and includes it
// if it is a valid IPv4 or IPv6 address.
func parseHostsIPs(b []byte) []string {
    seen := make(map[string]struct{})
    var out []string
    s := bufio.NewScanner(bytes.NewReader(b))
    for s.Scan() {
        line := strings.TrimSpace(s.Text())
        if line == "" || strings.HasPrefix(line, "#") { continue }
        // Remove inline comments
        if i := strings.Index(line, "#"); i >= 0 { line = strings.TrimSpace(line[:i]) }
        if line == "" { continue }
        // First column is whitespace-separated
        fields := strings.Fields(line)
        if len(fields) == 0 { continue }
        ip := fields[0]
        if v := net.ParseIP(ip); v != nil {
            if _, ok := seen[ip]; !ok {
                seen[ip] = struct{}{}
                out = append(out, ip)
            }
        }
    }
    return out
}

// discoverChildHosts runs a discovery command on the leader to fetch /etc/hosts,
// returning the raw content, exit code, and the parsed list of child host IPs.
func discoverChildHosts(client sessionClient, timeout time.Duration) (raw []byte, exitCode int, ips []string, err error) {
    out, code, e := runRemoteCommandFunc(client, "cat /etc/hosts", timeout)
    if e != nil {
        return out, code, nil, e
    }
    return out, code, parseHostsIPs(out), nil
}


package cmd

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestParseHostsIPs(t *testing.T) {
    data := []byte("# comment\n127.0.0.1 localhost\n10.0.0.1 host1 host1.alias # trailing\n10.0.0.1 dup\n::1 localhost\n\n192.168.1.5 host2\n")
    ips := parseHostsIPs(data)
    // Dedup preserves first occurrence order
    require.Equal(t, []string{"127.0.0.1", "10.0.0.1", "::1", "192.168.1.5"}, ips)
}


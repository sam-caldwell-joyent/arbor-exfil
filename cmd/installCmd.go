package cmd

import (
    "errors"
    "fmt"
    "io/fs"
    "os"
    "strings"

    "github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
    Use:   "install",
    Short: "Create arbor-exfil user and install SSH key on discovered hosts",
    RunE: func(cmd *cobra.Command, args []string) error {
        if cfgManifest == "" {
            return errors.New("--manifest is required (path to YAML)")
        }
        if cfgInstallPubKeyPath == "" {
            return errors.New("--install-pubkey is required (path to SSH public key)")
        }
        // Load manifest to source ssh_host defaults (leader connection)
        mf, err := loadManifest(cfgManifest)
        if err != nil {
            return fmt.Errorf("failed to read manifest: %w", err)
        }
        if cfgTarget == "" {
            if host := strings.TrimSpace(mf.SSHHost.IP); host != "" {
                if strings.Contains(host, ":") {
                    cfgTarget = host
                } else {
                    cfgTarget = host + ":22"
                }
            }
        }
        if cfgUser == "" {
            if u := strings.TrimSpace(mf.SSHHost.User); u != "" {
                cfgUser = u
            }
        }
        if cfgTarget == "" {
            return errors.New("--target is required (FQDN/IP:port)")
        }
        if cfgUser == "" {
            return errors.New("--user is required for SSH authentication")
        }

        // Read public key file
        pubBytes, err := os.ReadFile(cfgInstallPubKeyPath)
        if err != nil {
            if errors.Is(err, fs.ErrNotExist) {
                return fmt.Errorf("public key file not found: %s", cfgInstallPubKeyPath)
            }
            return fmt.Errorf("read public key: %w", err)
        }
        pubKey := strings.TrimSpace(string(pubBytes))

        // Connect to leader and perform discovery
        leaderClient, err := dialSSHFunc(cfgTarget, cfgUser, cfgPassword, cfgKeyPath, cfgPassphrase, cfgKnownHosts, cfgStrictHost, cfgConnTimeout)
        if err != nil {
            return fmt.Errorf("ssh connection to leader failed: %w", err)
        }
        defer func() { if leaderClient != nil { _ = leaderClient.Close() } }()

        // Establish a persistent shell for discovery
        var sessClient sessionClient
        if leaderClient != nil {
            ps, err := newPersistentShell(leaderClient)
            if err != nil { return fmt.Errorf("failed to start persistent session: %w", err) }
            defer func() { _ = ps.Close() }()
            sessClient = persistentSessionClient{ps}
        } else {
            sessClient = sshClientWrapper{leaderClient}
        }

        discOut, _, _, _ := discoverChildHosts(sessClient, cfgTimeout)
        hosts := parseHostsIPs(discOut)

        // Filter loopbacks (both IPv4 127.* and IPv6 ::1) for install
        filtered := make([]string, 0, len(hosts))
        for _, h := range hosts {
            if h == "::1" || strings.HasPrefix(h, "127.") { continue }
            filtered = append(filtered, h)
        }
        if len(filtered) == 0 {
            _, _ = fmt.Fprintln(os.Stderr, "No non-loopback hosts discovered; nothing to install")
            return nil
        }

        // Prepare the install script (will run via sudo -n /bin/sh -c ...)
        keyArg := shellQuote(pubKey)
        script := "u=arbor-exfil; " +
            "if ! id -u \"$u\" >/dev/null 2>&1; then (useradd -m -s /bin/bash \"$u\" 2>/dev/null || adduser --disabled-password --gecos '' \"$u\"); fi; " +
            "home=$(getent passwd \"$u\" | cut -d: -f6); [ -n \"$home\" ] || home=/home/$u; " +
            "install -d -m 700 -o \"$u\" -g \"$u\" \"$home/.ssh\"; " +
            fmt.Sprintf("printf '%%s\\n' %s >> \"$home/.ssh/authorized_keys\"; ", keyArg) +
            "chown -R \"$u\":\"$u\" \"$home/.ssh\"; chmod 600 \"$home/.ssh/authorized_keys\""

        full := fmt.Sprintf("sudo -n /bin/sh -c %s", shellQuote(script))

        // Iterate over hosts and perform installation
        var failures int
        for _, h := range filtered {
            target := h
            if !strings.Contains(h, ":") { target = h + ":22" }
            // Dial host
            client, err := dialSSHFunc(target, cfgUser, cfgPassword, cfgKeyPath, cfgPassphrase, cfgKnownHosts, cfgStrictHost, cfgConnTimeout)
            if err != nil {
                failures++
                _, _ = fmt.Fprintf(os.Stderr, "Host %s: dial failed: %v\n", target, err)
                continue
            }
            // Best-effort close
            func() {
                defer func() { if client != nil { _ = client.Close() } }()
                // Run the install command
                out, code, runErr := runRemoteCommandFunc(sshClientWrapper{client}, full, cfgTimeout)
                if runErr != nil || code != 0 {
                    failures++
                    if runErr != nil {
                        _, _ = fmt.Fprintf(os.Stderr, "Host %s: install error: %v\n", target, runErr)
                    } else {
                        _, _ = fmt.Fprintf(os.Stderr, "Host %s: install exit code %d\n", target, code)
                    }
                    if len(out) > 0 { _, _ = fmt.Fprintf(os.Stderr, "Output: %s\n", string(out)) }
                    return
                }
                _, _ = fmt.Fprintf(os.Stderr, "Host %s: arbor-exfil key installed\n", target)
            }()
        }
        if failures > 0 {
            return fmt.Errorf("install completed with %d failures", failures)
        }
        return nil
    },
}


package cmd

// manifest models the YAML schema consumed by arbor-exfil. It captures the
// report metadata, optional SSH defaults for the leader, and the list of
// commands to execute (which may be empty to request discovery-only).
type manifest struct {
    Name        string         `yaml:"name"`
    Description string         `yaml:"description"`
    SSHHost     sshHost        `yaml:"ssh_host,omitempty"`
    Commands    []commandEntry `yaml:"commands"`
}

// sshHost describes the remote leader connection details when not provided via
// CLI flags. CLI flags take precedence over these defaults when set.
type sshHost struct {
    IP   string `yaml:"ip"`
    User string `yaml:"user"`
}

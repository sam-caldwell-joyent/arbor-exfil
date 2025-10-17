package cmd

type manifest struct {
    Name        string         `yaml:"name"`
    Description string         `yaml:"description"`
    SSHHost     sshHost        `yaml:"ssh_host,omitempty"`
    Commands    []commandEntry `yaml:"commands"`
}

// sshHost describes the remote leader connection details when not
// provided via CLI flags. If CLI flags are set, they take precedence.
type sshHost struct {
    IP   string `yaml:"ip"`
    User string `yaml:"user"`
}

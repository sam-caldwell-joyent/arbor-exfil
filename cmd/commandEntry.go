package cmd

// commandEntry represents a single command specification from the manifest.
// It includes the base command, zero or more arguments (which are shell-quoted
// at execution time), an optional display title, the required shell path that
// should execute the command on the remote system, and an optional per-command
// timeout that overrides the global cmd-timeout flag.
type commandEntry struct {
    // "command" is preferred; "cmd" also accepted during unmarshal
    Command string   `yaml:"command"`
    Args    []string `yaml:"args"`
    // Optional display title for the section heading
    Title   string   `yaml:"title,omitempty"`
    // Optional shell override for this command (schema only; functionality TBD)
    Shell   string   `yaml:"shell,omitempty"`
    // Optional per-command timeout like "30s"; overrides global if set
    Timeout string `yaml:"timeout,omitempty"`
}

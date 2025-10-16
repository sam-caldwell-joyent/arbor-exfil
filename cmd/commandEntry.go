package cmd

type commandEntry struct {
    // "command" is preferred; "cmd" also accepted during unmarshal
    Command string   `yaml:"command"`
    Args    []string `yaml:"args"`
    // Optional display title for the section heading
    Title   string   `yaml:"title,omitempty"`
    // Optional per-command timeout like "30s"; overrides global if set
    Timeout string `yaml:"timeout,omitempty"`
}

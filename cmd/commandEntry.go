package cmd

type commandEntry struct {
	// "command" is preferred; "cmd" also accepted during unmarshal
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	// Optional per-command timeout like "30s"; overrides global if set
	Timeout string `yaml:"timeout,omitempty"`
}

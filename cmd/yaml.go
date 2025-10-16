package cmd

import (
    "fmt"
    "gopkg.in/yaml.v3"
)

// yamlUnmarshalImpl is separated for clarity/testability
func yamlUnmarshalImpl(b []byte, out any) error {
    if err := yaml.Unmarshal(b, out); err != nil {
        return fmt.Errorf("yaml unmarshal: %w", err)
    }
    return nil
}

// UnmarshalYAML supports both "command" and "cmd" keys for flexibility
func (c *commandEntry) UnmarshalYAML(value *yaml.Node) error {
    // Define a helper structure with both fields
    var aux struct {
        Command string   `yaml:"command"`
        Cmd     string   `yaml:"cmd"`
        Args    []string `yaml:"args"`
        Title   string   `yaml:"title"`
        Timeout string   `yaml:"timeout"`
    }
    if err := value.Decode(&aux); err != nil {
        return err
    }
    c.Command = aux.Command
    if c.Command == "" {
        c.Command = aux.Cmd
    }
    c.Args = aux.Args
    c.Title = aux.Title
    c.Timeout = aux.Timeout
    return nil
}

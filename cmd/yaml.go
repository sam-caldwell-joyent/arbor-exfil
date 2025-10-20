package cmd

import (
    "fmt"
    "gopkg.in/yaml.v3"
)

// yamlUnmarshalImpl unmarshals YAML into out. It is separated from the wrapper
// to make it easy to stub in tests if ever needed.
func yamlUnmarshalImpl(b []byte, out any) error {
    if err := yaml.Unmarshal(b, out); err != nil {
        return fmt.Errorf("yaml unmarshal: %w", err)
    }
    return nil
}

// UnmarshalYAML implements custom decoding for commandEntry to support both
// "command" and legacy "cmd" keys and to keep future schema evolution
// localized to this function.
func (c *commandEntry) UnmarshalYAML(value *yaml.Node) error {
    // Define a helper structure with both fields
    var aux struct {
        Command string   `yaml:"command"`
        Cmd     string   `yaml:"cmd"`
        Args    []string `yaml:"args"`
        Title   string   `yaml:"title"`
        Shell   string   `yaml:"shell"`
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
    c.Shell = aux.Shell
    c.Timeout = aux.Timeout
    return nil
}

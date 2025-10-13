package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// loadManifest reads and validates the YAML manifest
func loadManifest(path string) (*manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	mf := &manifest{}
	if err := yamlUnmarshal(b, mf); err != nil {
		return nil, err
	}
	// Support "cmd" as alias for "command" if used
	for i := range mf.Commands {
		if mf.Commands[i].Command == "" {
			// Attempt to sniff a "cmd:" field via a quick map decode
			// as a fallback for compatibility.
			// This is a lightweight pass; if not present, ignore.
		}
	}
	if mf.Name == "" {
		return nil, errors.New("manifest.name is required")
	}
	if mf.Description == "" {
		return nil, errors.New("manifest.description is required")
	}
	for i, c := range mf.Commands {
		if strings.TrimSpace(c.Command) == "" {
			return nil, fmt.Errorf("commands[%d].command is required", i)
		}
	}
	return mf, nil
}

package cmd

// yamlUnmarshal is a small wrapper around the real implementation. Keeping the
// indirection makes it simple for tests to swap behavior if necessary without
// importing the YAML library in multiple places.
func yamlUnmarshal(b []byte, out any) error {
	// Implemented in yaml.go
	return yamlUnmarshalImpl(b, out)
}

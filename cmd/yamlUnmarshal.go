package cmd

// yamlUnmarshal is a small wrapper to avoid importing yaml here and in tests
func yamlUnmarshal(b []byte, out any) error {
	// Implemented in yaml.go
	return yamlUnmarshalImpl(b, out)
}

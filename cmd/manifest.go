package cmd

type manifest struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Commands    []commandEntry `yaml:"commands"`
}

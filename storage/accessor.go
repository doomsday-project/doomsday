package storage

type Config struct {
	Type     string            `yaml:"type"`
	Address  string            `yaml:"address"`
	Auth     map[string]string `yaml:"auth"`
	BasePath string            `yaml:"base_path"`
}

type Accessor interface {
	List(path string) (PathList, error)
	Get(path string) (map[string]string, error)
}

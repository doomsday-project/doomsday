package storage

type Config struct {
	Type               string            `yaml:"type"`
	Address            string            `yaml:"address"`
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify"`
	Auth               map[string]string `yaml:"auth"`
	BasePath           string            `yaml:"base_path"`
	Name               string            `yaml:"name"`
}

type Accessor interface {
	List(path string) (PathList, error)
	Get(path string) (map[string]string, error)
}

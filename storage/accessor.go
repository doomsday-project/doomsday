package storage

type Config struct {
	Type               string            `yaml:"type"`
	Address            string            `yaml:"address"`
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify"`
	Auth               map[string]string `yaml:"auth"`
	Config             map[string]string `yaml:"config"`
}

type Accessor interface {
	List() (PathList, error)
	Get(path string) (map[string]string, error)
}

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type CLIConfig struct {
	Current string      `yaml:"current"`
	Targets []CLITarget `yaml:"targets"`
}

func (c CLIConfig) CurrentTarget() *CLITarget {
	return c.Find(c.Current)
}

func (c CLIConfig) Find(name string) *CLITarget {
	for _, target := range c.Targets {
		if target.Name == name {
			return &target
		}
	}

	return nil
}

func (c *CLIConfig) SetCurrent(name string) error {
	if c.Find(name) == nil {
		return fmt.Errorf("No target with name `%s' exists", name)
	}

	c.Current = name
	target = c.Find(c.Current)
	return nil
}

func (c *CLIConfig) Add(target CLITarget) error {
	for c.Find(target.Name) != nil {
		c.Delete(target.Name)
	}

	c.Targets = append(c.Targets, target)
	return nil
}

func (c *CLIConfig) Delete(name string) {
	for i, target := range c.Targets {
		if target.Name == name {
			c.Targets[i], c.Targets[len(c.Targets)-1] = c.Targets[len(c.Targets)-1], c.Targets[i]
			c.Targets = c.Targets[:len(c.Targets)-1]
			break
		}
	}
}

type CLITarget struct {
	Name       string `yaml:"name"`
	Address    string `yaml:"address"`
	Token      string `yaml:"token"`
	SkipVerify bool   `yaml:"skip_verify"`
}

func (c *CLITarget) String() string {
	return fmt.Sprintf(`Name: %s
Address: %s
SkipVerify: %t
`, c.Name, c.Address, c.SkipVerify)
}

func loadConfig(path string) (*CLIConfig, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("Could not open config at `%s': %s", path, err)
	}
	defer file.Close()

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Could not read from config (%s): %s", path, err)
	}

	conf := CLIConfig{}
	err = yaml.Unmarshal(fileContents, &conf)
	if err != nil {
		return nil, fmt.Errorf("Could not parse config (%s) as YAML: %s", path, err)
	}

	return &conf, nil
}

func (c *CLIConfig) saveConfig(path string) error {
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("Could not open config file at `%s' for writing: %s", path, err)
	}

	jEncoder := yaml.NewEncoder(file)
	err = jEncoder.Encode(&c)
	if err != nil {
		return fmt.Errorf("Could not write YAML to file at `%s': %s", path, err)
	}

	return nil
}

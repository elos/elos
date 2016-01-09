package command

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

const ConfigFileName = "elosconfig.json"

// The struct representing the state needed by the cli
type Config struct {
	Path string `json:"-"`
	Host string
}

// Read in the current configuration
func ParseConfigFile(path string) (*Config, error) {
	input, err := ioutil.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			c := Config{
				Path: path,
			}
			return &c, nil
		}
		return nil, err
	}

	c := Config{}
	if err := json.Unmarshal(input, &c); err != nil {
		return nil, err
	}

	c.Path = path

	return &c, nil
}

// Write out the configuration 'c'
func WriteConfigFile(c *Config) error {
	bytes, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	// user read write permissions
	return ioutil.WriteFile(c.Path, bytes, 0644)
}

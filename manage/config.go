// manage package

package manage

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// Config Constants
var (
	ConfigFilename = "manage.yaml"
)

type mon struct {
	Name       string `yaml:"name"`
	ConfigPath string `yaml:"config"`
}

// Config contains the parameters for Manage
type Config struct {
	Data     string `yaml:"data,omitempty"`
	Monitors []mon  `yaml:"monitors"`
}

// NewConfig creates a new Config
func NewConfig(configPath string) *Config {
	c := &Config{}
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		return nil
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Printf("Unmarshal: %v", err)
		return nil
	}
	return c
}

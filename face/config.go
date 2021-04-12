package face

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// Config contains the parameters for Face detection
type Config struct {
	Skip                    bool   `yaml:"skip,omitempty"`
	ForceCpu                bool   `yaml:"forceCpu,omitempty"`
	Padding                 int    `yaml:"padding,omitempty"`
	ModelFile               string `yaml:"modelFile,omitempty"`
	ConfigFile              string `yaml:"configFile,omitempty"`
	MinConfidencePercentage int    `yaml:"minConfidencePercentage,omitempty"`
	MaxPercentage           int    `yaml:"maxPercentage,omitempty"`
	MinOverlapPercentage    int    `yaml:"minOverlapPercentage,omitempty"`
	HighlightColor          string `yaml:"highlightColor,omitempty"`
	HighlightThickness      int    `yaml:"highlightThickness,omitempty"`
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

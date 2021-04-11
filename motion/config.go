package motion

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// Config contains the parameters for Motion detection
type Config struct {
	Skip               bool   `yaml:"skip,omitempty"`
	Padding            int    `yaml:"padding,omitempty"`
	MinimumPercentage  int    `yaml:"minPercentage,omitempty"`
	MaximumPercentage  int    `yaml:"maxPercentage,omitempty"`
	MaxMotions         int    `yaml:"maxMotions,omitempty"`
	ThresholdPercent   int    `yaml:"thresholdPercent,omitempty"`
	NoiseReduction     int    `yaml:"noiseReduction,omitempty"`
	HighlightColor     string `yaml:"highlightColor,omitempty"`
	HighlightThickness int    `yaml:"highlightThickness,omitempty"`
}

// NewConfig creates a new Config
func NewConfig(configPath string) *Config {
	c := &Config{
		MinimumPercentage: -1,
	}
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

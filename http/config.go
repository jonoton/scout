package http

import (
	"os"

	"github.com/jonoton/go-notify"
	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// Config Constants
var (
	ConfigFilename = "http.yaml"
)

// UserAuth contains the username, password, and optional two factor
type UserAuth struct {
	User      string          `yaml:"user"`
	Password  string          `yaml:"password"`
	TwoFactor notify.RxConfig `yaml:"twoFactor"`
}

type link struct {
	Name     string `yaml:"name,omitempty"`
	Url      string `yaml:"url"`
	User     string `yaml:"user,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// Config contains the parameters for Http
type Config struct {
	Port                int        `yaml:"port,omitempty"`
	LimitPerSecond      int        `yaml:"limitPerSecond,omitempty"`
	Users               []UserAuth `yaml:"users,omitempty"`
	SignInExpireDays    int        `yaml:"signInExpireDays,omitempty"`
	Links               []link     `yaml:"links,omitempty"`
	LinkRetry           int        `yaml:"linkRetry,omitempty"`
	TwoFactorTimeoutSec int        `yaml:"twoFactorTimeoutSec,omitempty"`
	LoginSigningKey     string     `yaml:"loginSigningKey,omitempty"`
}

// NewConfig creates a new Config
func NewConfig(configPath string) *Config {
	c := &Config{}
	yamlFile, err := os.ReadFile(configPath)
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

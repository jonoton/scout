package http

import (
	"os"

	"github.com/jonoton/go-notify"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

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
	LoginLimitPerSecond int        `yaml:"loginLimitPerSecond,omitempty"`
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

// IsBcryptHash checks if a string is already a bcrypt hash
func IsBcryptHash(s string) bool {
	return len(s) >= 4 && s[0:2] == "$2" && (s[2] == 'a' || s[2] == 'b' || s[2] == 'y') && s[3] == '$'
}

// IsSHA256Hash checks if a string is a 64-character hex string (SHA-256)
func IsSHA256Hash(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// SecureConfig updates the yaml config with hashed passwords
func (c *Config) SecureConfig(configPath string) error {
	changed := false
	for i := range c.Users {
		if !IsBcryptHash(c.Users[i].Password) {
			passToHash := getSHA256Hash(c.Users[i].Password)
			hash, err := bcrypt.GenerateFromPassword([]byte(passToHash), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			c.Users[i].Password = string(hash)
			changed = true
		}
	}

	for i := range c.Links {
		if c.Links[i].Password != "" && !IsSHA256Hash(c.Links[i].Password) {
			c.Links[i].Password = getSHA256Hash(c.Links[i].Password)
			changed = true
		}
	}

	if changed {
		data, err := yaml.Marshal(c)
		if err != nil {
			return err
		}
		err = os.WriteFile(configPath, data, 0600)
		if err != nil {
			return err
		}
		log.Printf("Updated %s with hashed passwords", configPath)
	}
	return nil
}

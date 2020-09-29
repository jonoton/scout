// notify package

package notify

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// Config Constants
var (
	SenderConfigFilename = "notify-sender.yaml"
	RxConfigFilename     = "notify-rx.yaml"
)

// SenderConfig contains parameters for notify sender
type SenderConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// NewSenderConfig creates a new SenderConfig
func NewSenderConfig(configPath string) *SenderConfig {
	n := &SenderConfig{}
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		return nil
	}
	err = yaml.Unmarshal(yamlFile, n)
	if err != nil {
		log.Printf("Unmarshal: %v", err)
		return nil
	}
	return n
}

// SmsConfig contains parameters for sms phone numbers
type SmsConfig struct {
	Verizon []string `yaml:"verizon,omitempty"`
	Att     []string `yaml:"att,omitempty"`
	Tmobile []string `yaml:"tmobile,omitempty"`
}

// RxConfig contains parameters for notifier receivers
type RxConfig struct {
	Email []string  `yaml:"email,omitempty"`
	Text  SmsConfig `yaml:"sms"`
}

// NewRxConfig returns a new RxConfig
func NewRxConfig(configPath string) *RxConfig {
	n := &RxConfig{}
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		return nil
	}
	err = yaml.Unmarshal(yamlFile, n)
	if err != nil {
		log.Printf("Unmarshal: %v", err)
		return nil
	}
	return n
}

// GetPhones returns a slice of Phone
func (c *RxConfig) GetPhones() []Phone {
	phones := []Phone{}
	for _, number := range c.Text.Verizon {
		phones = append(phones, *NewPhone(number, VERIZON))
	}
	for _, number := range c.Text.Att {
		phones = append(phones, *NewPhone(number, ATT))
	}
	for _, number := range c.Text.Tmobile {
		phones = append(phones, *NewPhone(number, TMOBILE))
	}
	return phones
}

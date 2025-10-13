package monitor

import (
	"os"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// Config contains the parameters for Monitor
type Config struct {
	Filename                   string `yaml:"filename,omitempty"`
	URL                        string `yaml:"url,omitempty"`
	MaxSourceFps               int    `yaml:"maxSourceFps,omitempty"`
	MaxOutputFps               int    `yaml:"maxOutputFps,omitempty"`
	Quality                    int    `yaml:"quality,omitempty"`
	CaptureTimeoutMilliSeconds int    `yaml:"captureTimeoutMilliSeconds,omitempty"`
	StaleTimeout               int    `yaml:"staleTimeout,omitempty"`
	StaleMaxRetry              int    `yaml:"staleMaxRetry,omitempty"`
	BufferSeconds              int    `yaml:"bufferSeconds,omitempty"`
	DelayBufferMilliSeconds    int    `yaml:"delayBufferMilliSeconds,omitempty"`
	MotionFilename             string `yaml:"motion,omitempty"`
	TensorFilename             string `yaml:"tensor,omitempty"`
	CaffeFilename              string `yaml:"caffe,omitempty"`
	FaceFilename               string `yaml:"face,omitempty"`
	NotifyRxFilename           string `yaml:"notifyRx,omitempty"`
	AlertFilename              string `yaml:"alert,omitempty"`
	RecordFilename             string `yaml:"record,omitempty"`
	ContinuousFilename         string `yaml:"continuous,omitempty"`
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

// RecordConfig contains the parameters for record settings
type RecordConfig struct {
	RecordObjects    bool   `yaml:"recordObjects,omitempty"`
	MaxPreSec        int    `yaml:"maxPreSec,omitempty"`
	TimeoutSec       int    `yaml:"timeoutSec,omitempty"`
	MaxSec           int    `yaml:"maxSec,omitempty"`
	DeleteAfterHours int    `yaml:"deleteAfterHours,omitempty"`
	DeleteAfterGB    int    `yaml:"deleteAfterGB,omitempty"`
	Codec            string `yaml:"codec,omitempty"`
	FileType         string `yaml:"fileType,omitempty"`
	BufferSeconds    int    `yaml:"bufferSeconds,omitempty"`
	PortableOnly     bool   `yaml:"portableOnly,omitempty"`
}

// NewRecordConfig creates a new RecordConfig
func NewRecordConfig(configPath string) *RecordConfig {
	c := &RecordConfig{}
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

// ContinuousConfig contains the parameters for record settings
type ContinuousConfig struct {
	TimeoutSec       int    `yaml:"timeoutSec,omitempty"`
	MaxSec           int    `yaml:"maxSec,omitempty"`
	DeleteAfterHours int    `yaml:"deleteAfterHours,omitempty"`
	DeleteAfterGB    int    `yaml:"deleteAfterGB,omitempty"`
	Codec            string `yaml:"codec,omitempty"`
	FileType         string `yaml:"fileType,omitempty"`
	BufferSeconds    int    `yaml:"bufferSeconds,omitempty"`
	PortableOnly     bool   `yaml:"portableOnly,omitempty"`
}

// NewContinuousConfig creates a new ContinuousConfig
func NewContinuousConfig(configPath string) *ContinuousConfig {
	c := &ContinuousConfig{}
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

// AlertConfig contains the parameters for alert notification settings
type AlertConfig struct {
	IntervalMinutes           int  `yaml:"intervalMinutes,omitempty"`
	MaxImagesPerInterval      int  `yaml:"maxImagesPerInterval,omitempty"`
	MaxSendAttachmentsPerHour int  `yaml:"maxSendAttachmentsPerHour,omitempty"`
	SaveQuality               int  `yaml:"saveQuality,omitempty"`
	SaveOriginal              bool `yaml:"saveOriginal,omitempty"`
	SaveHighlighted           bool `yaml:"saveHighlighted,omitempty"`
	SaveObjectsCount          int  `yaml:"saveObjectsCount,omitempty"`
	SaveFacesCount            int  `yaml:"saveFacesCount,omitempty"`
	TextAttachments           bool `yaml:"textAttachments,omitempty"`
	DeleteAfterHours          int  `yaml:"deleteAfterHours,omitempty"`
	DeleteAfterGB             int  `yaml:"deleteAfterGB,omitempty"`
}

// NewAlertConfig creates a new AlertConfig
func NewAlertConfig(configPath string) *AlertConfig {
	c := &AlertConfig{}
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

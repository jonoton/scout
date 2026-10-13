package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigDefaultCaptureTimeout(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	yamlContent := `
maxSourceFps: 10
maxOutputFps: 10
quality: 80
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	conf := NewConfig(configPath)
	if conf == nil {
		t.Fatalf("expected non-nil config")
	}

	if conf.CaptureTimeoutMilliSeconds != 3000 {
		t.Errorf("expected default CaptureTimeoutMilliSeconds to be 3000, got %d", conf.CaptureTimeoutMilliSeconds)
	}
}

func TestNewConfigCustomCaptureTimeout(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	yamlContent := `
maxSourceFps: 10
maxOutputFps: 10
captureTimeoutMilliSeconds: 5000
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	conf := NewConfig(configPath)
	if conf == nil {
		t.Fatalf("expected non-nil config")
	}

	if conf.CaptureTimeoutMilliSeconds != 5000 {
		t.Errorf("expected custom CaptureTimeoutMilliSeconds to be 5000, got %d", conf.CaptureTimeoutMilliSeconds)
	}
}

func TestNewConfigCaptureOptionsAndTolerance(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	yamlContent := `
url: "rtsp://10.0.0.1:554/stream"
maxSourceFps: 15
maxOutputFps: 15
captureOptions: "rtsp_transport;tcp|err_detect;explode"
captureMaxFailureTolerance: 5
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	conf := NewConfig(configPath)
	if conf == nil {
		t.Fatalf("expected non-nil config")
	}

	if conf.CaptureOptions != "rtsp_transport;tcp|err_detect;explode" {
		t.Errorf("expected CaptureOptions to be 'rtsp_transport;tcp|err_detect;explode', got %s", conf.CaptureOptions)
	}
	if conf.CaptureMaxFailureTolerance != 5 {
		t.Errorf("expected CaptureMaxFailureTolerance to be 5, got %d", conf.CaptureMaxFailureTolerance)
	}
}

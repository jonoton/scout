package manage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonoton/go-pubsubmutex"
	"github.com/jonoton/go-watcher"
	"github.com/jonoton/scout/monitor"
)

func TestDoCheckStaleMonitorsSetupFailedRemovesMonitor(t *testing.T) {
	m := &Manage{
		mons: make(monitor.Map),
		manageConf: Config{
			Monitors: []mon{
				{
					Name:       "offline-cam",
					ConfigPath: "/nonexistent-config.yaml",
				},
			},
		},
		wtr:    watcher.New(100 * time.Millisecond),
		pubsub: *pubsubmutex.NewPubSub(),
		cancel: make(chan bool),
		done:   make(chan bool),
	}

	mon := monitor.NewMonitor("offline-cam", nil)
	mon.IsStale = true
	mon.StaleMaxRetry = 3
	mon.StaleRetry = 3
	m.mons[mon.Name] = mon

	// Simulate stale check when config file does not exist (setupMonitor returns nil)
	m.doCheckStaleMonitors()

	// Since setupMonitor returns nil (config doesn't exist), monitor is removed
	if len(m.mons) != 0 {
		t.Errorf("Expected 0 active monitors after removal, got %d", len(m.mons))
	}
}

func TestDoCheckStaleMonitorsDecrementsRetryCount(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_cam.yaml")
	yamlContent := `
filename: "/nonexistent/test_video.mp4"
maxSourceFps: 10
maxOutputFps: 10
quality: 80
captureTimeoutMilliSeconds: 500
staleMaxRetry: 3
staleTimeout: 10
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	m := &Manage{
		mons: make(monitor.Map),
		manageConf: Config{
			Monitors: []mon{
				{
					Name:       "test-cam",
					ConfigPath: configPath,
				},
			},
		},
		wtr:    watcher.New(100 * time.Millisecond),
		pubsub: *pubsubmutex.NewPubSub(),
		cancel: make(chan bool),
		done:   make(chan bool),
	}

	mon := monitor.NewMonitor("test-cam", nil)
	mon.IsStale = true
	mon.StaleMaxRetry = 3
	mon.StaleRetry = 3
	m.mons[mon.Name] = mon

	m.doCheckStaleMonitors()

	newMon, ok := m.mons["test-cam"]
	if !ok {
		t.Fatalf("Expected test-cam to be restarted")
	}
	defer func() {
		newMon.Stop()
		newMon.Wait()
	}()

	if newMon.StaleRetry != 2 {
		t.Errorf("Expected StaleRetry to be decremented from 3 to 2, got %d", newMon.StaleRetry)
	}
}

func TestStaleRetryExhaustion(t *testing.T) {
	m := &Manage{
		mons:       make(monitor.Map),
		manageConf: Config{},
		wtr:        watcher.New(100 * time.Millisecond),
		pubsub:     *pubsubmutex.NewPubSub(),
		cancel:     make(chan bool),
		done:       make(chan bool),
	}

	mon := monitor.NewMonitor("dead-cam", nil)
	mon.IsStale = true
	mon.StaleMaxRetry = 3
	mon.StaleRetry = 0 // Exhausted!
	m.mons[mon.Name] = mon

	m.doCheckStaleMonitors()

	// Verify the exhausted monitor was stopped and removed, and not replaced
	if len(m.mons) != 0 {
		t.Errorf("Expected 0 active monitors when retry is exhausted, got %d", len(m.mons))
	}
}

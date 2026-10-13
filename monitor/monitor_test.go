package monitor

import (
	"testing"
	"time"

	"github.com/jonoton/go-videosource"
)

func TestMonitorStopNotStarted(t *testing.T) {
	mon := NewMonitor("test-unstarted", nil)

	done := make(chan bool)
	go func() {
		mon.Stop()
		mon.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Monitor.Stop() timed out when not started")
	}
}

func TestMonitorStopMultipleTimes(t *testing.T) {
	mon := NewMonitor("test-multi-stop", nil)

	done := make(chan bool)
	go func() {
		mon.Stop()
		mon.Stop()
		mon.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Monitor.Stop() timed out on multiple calls")
	}
}

func TestMonitorStopWithDelayBufferAndZeroFrames(t *testing.T) {
	reader := videosource.NewVideoReader(videosource.NewFileSource("offline", "/nonexistent/video.mp4"), 8, 8)
	mon := NewMonitor("test-offline-delay", reader)
	mon.SetBufferSeconds(2)
	mon.SetDelayBufferDuration(250)
	mon.SetStaleConfig(1, 3)
	tmpDir := t.TempDir()
	mon.SetRecord(tmpDir, &RecordConfig{
		RecordObjects:    true,
		MaxPreSec:        4,
		TimeoutSec:       10,
		MaxSec:           120,
		DeleteAfterHours: 72,
		DeleteAfterGB:    2,
	})
	mon.SetContinuous(tmpDir, &ContinuousConfig{
		TimeoutSec:       10,
		MaxSec:           10,
		DeleteAfterHours: 36,
		DeleteAfterGB:    2,
	})
	mon.SetAlert(nil, nil, tmpDir, &AlertConfig{
		IntervalMinutes:           4,
		MaxImagesPerInterval:      2,
		MaxSendAttachmentsPerHour: 5,
		SaveQuality:               75,
		DeleteAfterHours:          24,
		DeleteAfterGB:             2,
	})
	mon.Start()

	time.Sleep(200 * time.Millisecond)

	done := make(chan bool)
	go func() {
		mon.Stop()
		mon.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(4 * time.Second):
		t.Fatal("Monitor.Stop() and Wait() timed out on started monitor with delayBuffer and zero frames")
	}
}



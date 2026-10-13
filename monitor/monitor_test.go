package monitor

import (
	"testing"
	"time"
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

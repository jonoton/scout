package face

import (
	"testing"
	"time"
)

func TestFaceCloseNotStarted(t *testing.T) {
	face := NewFace("test-unstarted")

	done := make(chan bool)
	go func() {
		face.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Face.Close() timed out when not started")
	}
}

func TestFaceCloseMultipleTimes(t *testing.T) {
	face := NewFace("test-multi-close")

	done := make(chan bool)
	go func() {
		face.Close()
		face.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Face.Close() timed out on multiple calls")
	}
}

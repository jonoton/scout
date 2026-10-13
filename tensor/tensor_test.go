package tensor

import (
	"testing"
	"time"
)

func TestTensorCloseNotStarted(t *testing.T) {
	tensor := NewTensor("test-unstarted")

	done := make(chan bool)
	go func() {
		tensor.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Tensor.Close() timed out when not started")
	}
}

func TestTensorCloseMultipleTimes(t *testing.T) {
	tensor := NewTensor("test-multi-close")

	done := make(chan bool)
	go func() {
		tensor.Close()
		tensor.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Tensor.Close() timed out on multiple calls")
	}
}

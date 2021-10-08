package videosource

import (
	"time"
)

type FrameStats struct {
	AcceptedTotal     int
	AcceptedPerSecond int
	DroppedTotal      int
	DroppedPerSecond  int
}

// VideoStats contains video statistics
type VideoStats struct {
	AcceptedTotal     int
	AcceptedPerSecond int
	DroppedTotal      int
	DroppedPerSecond  int
	acceptedTmp       int
	droppedTmp        int
	stop              chan bool
}

// NewVideoStats creates a new VideoStats
func NewVideoStats() *VideoStats {
	v := &VideoStats{
		stop: make(chan bool),
	}
	return v
}

func (v *VideoStats) Start() {
	go func() {
		fpsTick := time.NewTicker(time.Second)
	Loop:
		for {
			select {
			case _, ok := <-fpsTick.C:
				if !ok {
					break Loop
				}
				v.AcceptedPerSecond = v.acceptedTmp
				v.acceptedTmp = 0
				v.DroppedPerSecond = v.droppedTmp
				v.droppedTmp = 0
			case <-v.stop:
				break Loop
			}
		}
		fpsTick.Stop()
	}()
}

// AddAccepted adds an accepted image
func (v *VideoStats) AddAccepted() {
	v.AcceptedTotal++
	v.acceptedTmp++
}

// AddDropped adds a dropped image
func (v *VideoStats) AddDropped() {
	v.DroppedTotal++
	v.droppedTmp++
}

// GetStats returns the FrameStats
func (v *VideoStats) GetStats() FrameStats {
	return FrameStats{
		AcceptedTotal:     v.AcceptedTotal,
		AcceptedPerSecond: v.AcceptedPerSecond,
		DroppedTotal:      v.DroppedTotal,
		DroppedPerSecond:  v.DroppedPerSecond,
	}
}

// Cleanup the VideoStats
func (v *VideoStats) Cleanup() {
	v.AcceptedPerSecond = 0
	v.DroppedPerSecond = 0
	close(v.stop)
}

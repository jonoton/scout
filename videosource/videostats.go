package videosource

import (
	"time"
)

// VideoStats contains video statistics
type VideoStats struct {
	AcceptedTotal     int
	AcceptedPerSecond int
	DroppedTotal      int
	DroppedPerSecond  int
	acceptedTmp       int
	droppedTmp        int
	fpsTick           *time.Ticker
}

// NewVideoStats creates a new VideoStats
func NewVideoStats() *VideoStats {
	v := &VideoStats{
		fpsTick: time.NewTicker(time.Second),
	}
	go func() {
		for {
			select {
			case _, ok := <-v.fpsTick.C:
				if !ok {
					return
				}
				v.AcceptedPerSecond = v.acceptedTmp
				v.acceptedTmp = 0
				v.DroppedPerSecond = v.droppedTmp
				v.droppedTmp = 0
			}
		}
	}()
	return v
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

// Cleanup the VideoStats
func (v *VideoStats) Cleanup() {
	v.AcceptedPerSecond = 0
	v.DroppedPerSecond = 0
	if v.fpsTick != nil {
		v.fpsTick.Stop()
	}
}

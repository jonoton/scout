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
	stop              chan bool
}

// NewVideoStats creates a new VideoStats
func NewVideoStats() *VideoStats {
	v := &VideoStats{
		fpsTick: time.NewTicker(time.Second),
		stop:    make(chan bool),
	}
	go func() {
	Loop:
		for {
			select {
			case _, ok := <-v.fpsTick.C:
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
		v.fpsTick.Stop()
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
		close(v.stop)
	}
}

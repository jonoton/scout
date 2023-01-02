package videosource

type FrameStats struct {
	AcceptedTotal     int
	AcceptedPerSecond int
	DroppedTotal      int
	DroppedPerSecond  int
}

type FrameStatsCombo struct {
	In  FrameStats
	Out FrameStats
}

// VideoStats contains video statistics
type VideoStats struct {
	acceptedTotal     int
	acceptedPerSecond int
	droppedTotal      int
	droppedPerSecond  int
	acceptedTmp       int
	droppedTmp        int
}

// NewVideoStats creates a new VideoStats
func NewVideoStats() *VideoStats {
	v := &VideoStats{}
	return v
}

// Tick every second
func (v *VideoStats) Tick() {
	v.acceptedPerSecond = v.acceptedTmp
	v.acceptedTmp = 0
	v.droppedPerSecond = v.droppedTmp
	v.droppedTmp = 0
}

// AddAccepted adds an accepted image
func (v *VideoStats) AddAccepted() {
	v.acceptedTotal++
	v.acceptedTmp++
}

// AddDropped adds a dropped image
func (v *VideoStats) AddDropped() {
	v.droppedTotal++
	v.droppedTmp++
}

// GetStats returns the FrameStats
func (v *VideoStats) GetStats() *FrameStats {
	f := &FrameStats{
		AcceptedTotal:     v.acceptedTotal,
		AcceptedPerSecond: v.acceptedPerSecond,
		DroppedTotal:      v.droppedTotal,
		DroppedPerSecond:  v.droppedPerSecond,
	}
	return f
}

// ClearPerSecond the VideoStats
func (v *VideoStats) ClearPerSecond() {
	v.acceptedPerSecond = 0
	v.droppedPerSecond = 0
}

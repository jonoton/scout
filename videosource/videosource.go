package videosource

// VideoSource interface for setting up and reading images
type VideoSource interface {
	GetName() string
	Initialize() (ok bool)
	Cleanup()
	ReadImage() (done bool, image Image)
}

// BaseVideo contains common video source info
type BaseVideo struct {
	name string
}

// NewBaseVideo creates a new BaseVideo
func NewBaseVideo(name string) *BaseVideo {
	b := &BaseVideo{
		name: name,
	}
	return b
}

// GetName implements interface
func (b *BaseVideo) GetName() string {
	return b.name
}

// Initialize implements interface
func (b *BaseVideo) Initialize() (ok bool) {
	// implement in source type
	return
}

// Cleanup implements interface
func (b *BaseVideo) Cleanup() {
	// implement in source type
}

// ReadImage implements interface
func (b *BaseVideo) ReadImage() (done bool, image Image) {
	// implement in source type
	return
}

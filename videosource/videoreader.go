package videosource

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// VideoReader reads a VideoSource
type VideoReader struct {
	videoSource  VideoSource
	SourceStats  *VideoStats
	OutputStats  *VideoStats
	done         chan bool
	cancel       chan bool
	MaxSourceFps int
	MaxOutputFps int
	Quality      int
}

// NewVideoReader creates a new VideoReader
func NewVideoReader(videoSource VideoSource, maxSourceFps int, maxOutputFps int) *VideoReader {
	if videoSource == nil || maxSourceFps <= 0 || maxOutputFps <= 0 {
		return nil
	}
	v := &VideoReader{
		videoSource:  videoSource,
		SourceStats:  NewVideoStats(),
		OutputStats:  NewVideoStats(),
		done:         make(chan bool),
		cancel:       make(chan bool),
		MaxSourceFps: maxSourceFps,
		MaxOutputFps: maxOutputFps,
		Quality:      100,
	}
	return v
}

// SetQuality sets the Image quality
func (v *VideoReader) SetQuality(percent int) {
	if percent > 0 && percent < 100 {
		v.Quality = percent
	}
}

// Start runs the processes
func (v *VideoReader) Start() <-chan Image {
	images := make(chan Image)
	go func() {
		if !v.videoSource.Initialize() {
			log.Warnln("VideoReader could not initialize", v.videoSource.GetName())
		}
		videoImgs := v.sourceImages()
		var bufImage *Image
		fps := v.MaxOutputFps
		outTick := time.NewTicker(v.getTickMs(fps) * time.Millisecond)
	Loop:
		for {
			select {
			case img, ok := <-videoImgs:
				if !ok {
					img.Cleanup()
					break Loop
				}
				if bufImage != nil {
					if filled, closed := bufImage.Cleanup(); filled && closed {
						v.OutputStats.AddDropped()
					}
				}
				bufImage = &img
			case <-outTick.C:
				if bufImage != nil && bufImage.IsFilled() {
					images <- *bufImage.Ref()
					bufImage.Cleanup()
					bufImage = nil
					v.OutputStats.AddAccepted()
				}
				if fps != v.MaxOutputFps {
					fps = v.MaxOutputFps
					outTick.Stop()
					outTick = time.NewTicker(v.getTickMs(fps) * time.Millisecond)
				}
			}
		}
		if bufImage != nil {
			bufImage.Cleanup()
		}
		outTick.Stop()
		v.OutputStats.Cleanup()
		v.videoSource.Cleanup()
		close(images)
		close(v.done)
	}()

	return images
}

// Stop will stop the processes
func (v *VideoReader) Stop() {
	close(v.cancel)
}

// Wait for done
func (v *VideoReader) Wait() {
	<-v.done
}

func (v *VideoReader) getTickMs(fps int) time.Duration {
	tickMs := 5
	if fps > 0 {
		tickMs = 1000 / fps
	}
	return time.Duration(tickMs)
}

func (v *VideoReader) sourceImages() <-chan Image {
	videoImgs := make(chan Image)
	go func() {
		fps := v.MaxSourceFps
		tick := time.NewTicker(v.getTickMs(fps) * time.Millisecond)
	Loop:
		for {
			select {
			case <-tick.C:
				done, image := v.videoSource.ReadImage()
				if done {
					image.Cleanup()
					log.Infoln("Done source", v.videoSource.GetName())
					break Loop
				} else if image.IsFilled() {
					if v.Quality > 0 && v.Quality < 100 {
						image.ChangeQuality(v.Quality)
					}
					videoImgs <- *image.Ref()
					v.SourceStats.AddAccepted()
				}
				if fps != v.MaxSourceFps {
					fps = v.MaxSourceFps
					tick.Stop()
					tick = time.NewTicker(v.getTickMs(fps) * time.Millisecond)
				}
				image.Cleanup()
			case <-v.cancel:
				break Loop
			}
		}
		tick.Stop()
		v.SourceStats.Cleanup()
		close(videoImgs)
	}()
	return videoImgs
}

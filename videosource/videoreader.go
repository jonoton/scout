package videosource

import (
	"time"

	"github.com/cskr/pubsub"
	pubsubmutex "github.com/jonoton/go-pubsubmutex"
	log "github.com/sirupsen/logrus"
)

const topicGetFrameStatsSource = "topic-get-frame-stats-source"
const topicCurrentFrameStatsSource = "topic-current-frame-stats-source"
const topicGetFrameStatsOutput = "topic-get-frame-stats-output"
const topicCurrentFrameStatsOutput = "topic-current-frame-stats-output"

// VideoReader reads a VideoSource
type VideoReader struct {
	videoSource  VideoSource
	pubsubSource pubsubmutex.PubSubMutex
	pubsubOutput pubsubmutex.PubSubMutex
	sourceStats  VideoStats
	outputStats  VideoStats
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
		pubsubSource: *pubsubmutex.New(0),
		pubsubOutput: *pubsubmutex.New(0),
		sourceStats:  *NewVideoStats(),
		outputStats:  *NewVideoStats(),
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
	v.pubsubSource.Start()
	v.pubsubOutput.Start()
	go func() {
		if !v.videoSource.Initialize() {
			log.Warnln("VideoReader could not initialize", v.videoSource.GetName())
		}
		videoImgs := v.sourceImages()
		var bufImage *Image
		fps := v.MaxOutputFps
		outTick := time.NewTicker(v.getTickMs(fps) * time.Millisecond)
		statTick := time.NewTicker(time.Second)
		getFrameStatsChan := v.pubsubOutput.Sub(topicGetFrameStatsOutput)
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
						v.outputStats.AddDropped()
					}
				}
				bufImage = &img
			case <-outTick.C:
				if bufImage != nil && bufImage.IsFilled() {
					images <- *bufImage.Ref()
					bufImage.Cleanup()
					bufImage = nil
					v.outputStats.AddAccepted()
				}
				if fps != v.MaxOutputFps {
					fps = v.MaxOutputFps
					outTick.Stop()
					outTick = time.NewTicker(v.getTickMs(fps) * time.Millisecond)
				}
			case <-statTick.C:
				v.outputStats.Tick()
				v.pubOutputStats()
			case _, ok := <-getFrameStatsChan:
				if !ok {
					continue
				}
				v.pubOutputStats()
			}
		}
		if bufImage != nil {
			bufImage.Cleanup()
		}
		outTick.Stop()
		statTick.Stop()
		v.videoSource.Cleanup()
		v.outputStats.ClearPerSecond()
		v.pubsubSource.Shutdown()
		v.pubsubOutput.Shutdown()
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
		statTick := time.NewTicker(time.Second)
		getFrameStatsChan := v.pubsubSource.Sub(topicGetFrameStatsSource)
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
					v.sourceStats.AddAccepted()
				}
				if fps != v.MaxSourceFps {
					fps = v.MaxSourceFps
					tick.Stop()
					tick = time.NewTicker(v.getTickMs(fps) * time.Millisecond)
				}
				image.Cleanup()
			case <-statTick.C:
				v.sourceStats.Tick()
				v.pubSourceStats()
			case _, ok := <-getFrameStatsChan:
				if !ok {
					continue
				}
				v.pubSourceStats()
			case <-v.cancel:
				break Loop
			}
		}
		tick.Stop()
		statTick.Stop()
		v.sourceStats.ClearPerSecond()
		close(videoImgs)
	}()
	return videoImgs
}

// GetStatsSource returns the FrameStats
func (v *VideoReader) GetStatsSource(timeoutMs int) (result *FrameStats) {
	r := v.pubsubSource.SendReceive(topicGetFrameStatsSource, topicCurrentFrameStatsSource,
		nil, timeoutMs)
	if r != nil {
		result = r.(*FrameStats)
	}
	return
}
func (v *VideoReader) pubSourceStats() {
	v.pubsubSource.Use(func(instance *pubsub.PubSub) {
		instance.TryPub(v.sourceStats.GetStats(), topicCurrentFrameStatsSource)
	})
}

// GetSourceStatsChan returns the channel
func (v *VideoReader) GetSourceStatsChan() (result <-chan interface{}) {
	result = v.pubsubSource.SubAsync(topicCurrentFrameStatsSource)
	return
}

// GetStatsOutput returns the FrameStats
func (v *VideoReader) GetStatsOutput(timeoutMs int) (result *FrameStats) {
	r := v.pubsubOutput.SendReceive(topicGetFrameStatsOutput, topicCurrentFrameStatsOutput,
		nil, timeoutMs)
	if r != nil {
		result = r.(*FrameStats)
	}
	return
}
func (v *VideoReader) pubOutputStats() {
	v.pubsubOutput.Use(func(instance *pubsub.PubSub) {
		instance.TryPub(v.outputStats.GetStats(), topicCurrentFrameStatsOutput)
	})
}

// GetOutputStatsChan returns the channel
func (v *VideoReader) GetOutputStatsChan() (result <-chan interface{}) {
	result = v.pubsubOutput.SubAsync(topicCurrentFrameStatsOutput)
	return
}

package videosource

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cskr/pubsub"
	pubsubmutex "github.com/jonoton/go-pubsubmutex"
	"github.com/jonoton/go-sharedmat"
	log "github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
)

const topicGetFrameStats = "topic-get-frame-stats"
const topicCurrentFrameStats = "topic-current-frame-stats"

// SaveImage will save an Image
func SaveImage(img Image, t time.Time, saveDirectory string, jpegQuality int, name string, title string, percentage string) (savePath string) {
	savePath = GetImageFilename(t, saveDirectory, name, title, percentage)
	jpgParams := []int{gocv.IMWriteJpegQuality, jpegQuality}
	if img.SharedMat != nil {
		img.SharedMat.Guard.RLock()
		if sharedmat.Filled(&img.SharedMat.Mat) {
			gocv.IMWriteWithParams(savePath, img.SharedMat.Mat, jpgParams)
		}
		img.SharedMat.Guard.RUnlock()
	}
	return
}

// SavePreview will save a smaller Image
func SavePreview(img Image, t time.Time, saveDirectory string, name string, title string, percentage string) (savePath string) {
	savePath = GetImageFilename(t, saveDirectory, name, title, percentage)
	scaledImage := img.ScaleToWidth(128)
	if scaledImage.SharedMat != nil {
		scaledImage.SharedMat.Guard.RLock()
		if sharedmat.Filled(&scaledImage.SharedMat.Mat) {
			gocv.IMWrite(savePath, scaledImage.SharedMat.Mat)
		}
		scaledImage.SharedMat.Guard.RUnlock()
	}
	scaledImage.Cleanup()
	os.Rename(savePath, savePath+".preview")
	return
}

// VideoWriter constants
const (
	ActivityImage  = 0
	ActivityMotion = 1
	ActivityObject = 2
	ActivityFace   = 3
)

// VideoWriter writes Images
type VideoWriter struct {
	record           bool
	recording        bool
	startTime        time.Time
	name             string
	saveDirectory    string
	codec            string
	fileType         string
	maxSec           int
	outFps           int
	streamChan       chan ProcessedImage
	preRingBuffer    RingBufferImage
	writerFull       *gocv.VideoWriter
	writerPortable   *gocv.VideoWriter
	activitySec      int
	lastActivityTime time.Time
	secTick          *time.Ticker
	recordChan       chan bool
	done             chan bool
	videoStats       *VideoStats
	savePreview      bool
	savePortable     bool
	saveFull         bool
	activityType     int
	PortableWidth    int
	pubsub           pubsubmutex.PubSubMutex
}

// NewVideoWriter creates a new VideoWriter
func NewVideoWriter(name string, saveDirectory string, codec string, fileType string, bufferSeconds int,
	maxPreSec int, timeoutSec int, maxSec int, outFps int, savePreview bool, savePortable bool, saveFull bool, activityType int) *VideoWriter {
	if saveDirectory == "" || codec == "" || fileType == "" || timeoutSec <= 0 || maxSec <= 0 || outFps <= 0 {
		return nil
	}
	bufferSize := bufferSeconds * outFps
	if bufferSize < 0 {
		bufferSize = 0
	}
	preRingBufferSize := maxPreSec * outFps
	if preRingBufferSize <= 0 {
		preRingBufferSize = 1
	}
	v := &VideoWriter{
		record:           false,
		recording:        false,
		startTime:        time.Time{},
		name:             name,
		saveDirectory:    saveDirectory,
		codec:            codec, // xvid, mp4v, hev1, hvc1, avc1, avc3
		fileType:         fileType,
		maxSec:           maxSec,
		outFps:           outFps,
		streamChan:       make(chan ProcessedImage, bufferSize),
		preRingBuffer:    *NewRingBufferImage(preRingBufferSize),
		writerFull:       nil,
		writerPortable:   nil,
		activitySec:      timeoutSec,
		lastActivityTime: time.Time{},
		secTick:          time.NewTicker(time.Second),
		recordChan:       make(chan bool),
		done:             make(chan bool),
		videoStats:       NewVideoStats(),
		savePreview:      savePreview,
		savePortable:     savePortable,
		saveFull:         saveFull,
		activityType:     activityType,
		PortableWidth:    1080,
		pubsub:           *pubsubmutex.New(0),
	}
	return v
}

// Start runs the processes
func (v *VideoWriter) Start() {
	v.pubsub.Start()
	go func() {
		statTick := time.NewTicker(time.Second)
		getFrameStatsChan := v.pubsub.Sub(topicGetFrameStats)
	Loop:
		for {
			select {
			case <-statTick.C:
				v.videoStats.Tick()
				v.pubStats()
			case _, ok := <-getFrameStatsChan:
				if !ok {
					continue
				}
				v.pubStats()
			case <-v.recordChan:
				v.record = true
				v.lastActivityTime = time.Now()
			case img, ok := <-v.streamChan:
				if !ok {
					if v.recording {
						// close
						v.closeRecord()
					}
					img.Cleanup()
					break Loop
				}
				if v.record &&
					((v.activityType == ActivityImage && img.Original.IsFilled()) ||
						(v.activityType == ActivityMotion && img.HasMotion()) ||
						(v.activityType == ActivityObject && img.HasObject()) ||
						(v.activityType == ActivityFace && img.HasFace())) {
					v.lastActivityTime = time.Now()
				}
				origImg := *img.Original.Ref()
				if origImg.IsFilled() {
					if v.recording {
						// write
						v.writeRecord(origImg)
						v.videoStats.AddAccepted()
					} else {
						// buffer
						oldest := v.preRingBuffer.Push(*origImg.Ref())
						if filled, closed := oldest.Cleanup(); filled && closed {
							v.videoStats.AddDropped()
						}
					}
				}
				origImg.Cleanup()
				if v.record && !v.recording {
					// open
					firstFrame := v.preRingBuffer.Pop()
					if firstFrame.IsFilled() {
						popped := v.preRingBuffer.PopAll()
						preFrames := *NewImageList()
						preFrames.Set(popped)
						preview := firstFrame
						if len(popped) > 0 {
							preview = popped[0]
						}
						v.openRecord(firstFrame, preview)
						v.writeRecord(firstFrame)
						firstFrame.Cleanup()
						for preFrames.Len() > 0 {
							cur := preFrames.Pop()
							v.writeRecord(cur)
							cur.Cleanup()
						}
					}
				} else if !v.record && v.recording {
					// close
					v.closeRecord()
				}
				img.Cleanup()
			case <-v.secTick.C:
				if v.isRecordExpired() {
					v.closeRecord()
				}
				if v.isActivityExpired() {
					v.record = false
					v.lastActivityTime = time.Time{}
					v.closeRecord()
				}
			}
		}
		statTick.Stop()
		v.secTick.Stop()
		v.videoStats.ClearPerSecond()
		v.pubsub.Shutdown()
		cleanupRingBuffer(&v.preRingBuffer)
		close(v.done)
	}()
}

// Trigger to start recording
func (v *VideoWriter) Trigger() {
	v.recordChan <- true
}

// Send Image to write
func (v *VideoWriter) Send(img ProcessedImage) {
	v.streamChan <- img
}

// Close notified by caller that input stream is done/closed
func (v *VideoWriter) Close() {
	close(v.streamChan)
}

// Wait until done
func (v *VideoWriter) Wait() {
	<-v.done
}

func (v *VideoWriter) openRecord(img Image, preview Image) {
	timeNow := time.Now()
	if v.saveFull {
		saveFilenameFull := GetVideoFilename(timeNow, v.saveDirectory, v.name, v.fileType, false)
		wFull, err := gocv.VideoWriterFile(saveFilenameFull,
			v.codec, float64(v.outFps),
			img.Width(), img.Height(), true)
		if err == nil {
			v.recording = true
			v.writerFull = wFull

		} else {
			log.Error("Could not open gocv writer full")
		}
	}
	if v.savePortable {
		scaledImage := img.ScaleToWidth(v.PortableWidth)
		saveFilenamePortable := GetVideoFilename(timeNow, v.saveDirectory, v.name, v.fileType, true)
		wPortable, err := gocv.VideoWriterFile(saveFilenamePortable,
			v.codec, float64(v.outFps),
			scaledImage.Width(), scaledImage.Height(), true)
		if err == nil {
			v.recording = true
			v.writerPortable = wPortable
		} else {
			log.Error("Could not open gocv writer portable")
		}
		scaledImage.Cleanup()
	}
	if v.recording {
		v.startTime = timeNow
		if v.savePreview {
			SavePreview(preview, timeNow, v.saveDirectory, v.name, "", "")
		}
	}
}

func (v *VideoWriter) closeRecord() {
	if v.writerFull != nil {
		v.writerFull.Close()
	}
	if v.writerPortable != nil {
		v.writerPortable.Close()
	}
	v.startTime = time.Time{}
	v.recording = false
}

func cleanupRingBuffer(ringBuffer *RingBufferImage) {
	for ringBuffer.Len() > 0 {
		oldest := ringBuffer.Pop()
		oldest.Cleanup()
	}
}

func (v *VideoWriter) isRecordExpired() bool {
	return !v.startTime.IsZero() && time.Since(v.startTime) > (time.Duration(v.maxSec)*time.Second)
}

func (v *VideoWriter) isActivityExpired() bool {
	return !v.lastActivityTime.IsZero() && time.Since(v.lastActivityTime) > (time.Duration(v.activitySec)*time.Second)
}

func (v *VideoWriter) writeRecord(img Image) {
	if v.writerFull != nil {
		if img.SharedMat != nil {
			img.SharedMat.Guard.RLock()
			if sharedmat.Filled(&img.SharedMat.Mat) {
				v.writerFull.Write(img.SharedMat.Mat)
			}
			img.SharedMat.Guard.RUnlock()
		}
	}
	if v.writerPortable != nil {
		scaledImage := img.ScaleToWidth(v.PortableWidth)
		if scaledImage.SharedMat != nil {
			scaledImage.SharedMat.Guard.RLock()
			if sharedmat.Filled(&scaledImage.SharedMat.Mat) {
				v.writerPortable.Write(scaledImage.SharedMat.Mat)
			}
			scaledImage.SharedMat.Guard.RUnlock()
		}
		scaledImage.Cleanup()
	}
}

// GetBaseFilename will return a formatted base filename with current date time
func GetBaseFilename(t time.Time, saveDirectory string, name string, title string, percentage string) string {
	filename := path.Join(saveDirectory, name)
	filename += "_" + fmt.Sprintf("%d_%02d_%02d_%02d_%02d_%02d_%09d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond())
	if len(title) > 0 {
		filename += "_" + title
	}
	if len(percentage) > 0 {
		filename += "_" + percentage
	}
	return filename
}

// GetVideoFilename will return a video filename
func GetVideoFilename(t time.Time, saveDirectory string, name string, fileType string, portable bool) (filename string) {
	if portable {
		filename = GetBaseFilename(t, saveDirectory, name, "Portable", "")
	} else {
		filename = GetBaseFilename(t, saveDirectory, name, "Full", "")
	}
	filename += "." + strings.ToLower(fileType)
	return filename
}

// GetImageFilename will return an image filename
func GetImageFilename(t time.Time, saveDirectory string, name string, title string, percentage string) string {
	filename := GetBaseFilename(t, saveDirectory, name, title, percentage)
	filename += ".jpg"
	return filename
}

// GetStats returns the FrameStats
func (v *VideoWriter) GetStats(timeoutMs int) (result *FrameStats) {
	r := v.pubsub.SendReceive(topicGetFrameStatsOutput, topicCurrentFrameStats,
		nil, timeoutMs)
	if r != nil {
		result = r.(*FrameStats)
	}
	return
}
func (v *VideoWriter) pubStats() {
	v.pubsub.Use(func(instance *pubsub.PubSub) {
		instance.TryPub(v.videoStats.GetStats(), topicCurrentFrameStats)
	})
}

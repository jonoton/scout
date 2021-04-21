package videosource

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jonoton/scout/sharedmat"
	log "github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
)

// SaveImage will save an Image
func SaveImage(img Image, t time.Time, saveDirectory string, jpegQuality int, name string, title string, percentage string) (savePath string) {
	savePath = GetImageFilename(t, saveDirectory, name, title, percentage)
	jpgParams := []int{gocv.IMWriteJpegQuality, jpegQuality}
	if img.SharedMat != nil {
		img.SharedMat.Guard.RLock()
		if sharedmat.Valid(&img.SharedMat.Mat) {
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
		if sharedmat.Valid(&scaledImage.SharedMat.Mat) {
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
	ActivityObject = 0
	ActivityFace   = 1
)

// VideoWriter writes Images
type VideoWriter struct {
	Record         bool
	recording      bool
	startTime      time.Time
	name           string
	saveDirectory  string
	codec          string
	fileType       string
	timeoutSec     int
	maxSec         int
	outFps         int
	streamChan     ProcessedImageFpsChan
	preRingBuffer  RingBufferImage
	writerFull     *gocv.VideoWriter
	writerPortable *gocv.VideoWriter
	timeoutTick    *time.Ticker
	secTick        *time.Ticker
	activity       bool
	done           chan bool
	VideoStats     *VideoStats
	savePreview    bool
	savePortable   bool
	activityType   int
	PortableWidth  int
}

// NewVideoWriter creates a new VideoWriter
func NewVideoWriter(name string, saveDirectory string, codec string, fileType string,
	maxPreSec int, timeoutSec int, maxSec int, outFps int, savePreview bool, savePortable bool, activityType int) *VideoWriter {
	if saveDirectory == "" || codec == "" || fileType == "" || timeoutSec <= 0 || maxSec <= 0 || outFps <= 0 {
		return nil
	}
	v := &VideoWriter{
		Record:         false,
		recording:      false,
		startTime:      time.Time{},
		name:           name,
		saveDirectory:  saveDirectory,
		codec:          codec,
		fileType:       fileType,
		timeoutSec:     timeoutSec,
		maxSec:         maxSec,
		outFps:         outFps,
		streamChan:     *NewProcessedImageFpsChan(outFps),
		preRingBuffer:  *NewRingBufferImage(maxPreSec * outFps),
		writerFull:     nil,
		writerPortable: nil,
		timeoutTick:    time.NewTicker(time.Duration(timeoutSec/2) * time.Second),
		secTick:        time.NewTicker(time.Second),
		activity:       false,
		done:           make(chan bool),
		VideoStats:     NewVideoStats(),
		savePreview:    savePreview,
		savePortable:   savePortable,
		activityType:   activityType,
		PortableWidth:  1080,
	}
	return v
}

// Start runs the processes
func (v *VideoWriter) Start() {
	go func() {
		defer close(v.done)
		defer cleanupRingBuffer(&v.preRingBuffer)
		streamChan := v.streamChan.Start()
		for {
			select {
			case img, ok := <-streamChan:
				if !ok {
					if v.recording {
						// close
						v.closeRecord()
					}
					img.Cleanup()
					return
				}
				if (v.activityType == ActivityObject && len(img.Objects) > 0) ||
					(v.activityType == ActivityFace && len(img.Faces) > 0) {
					v.activity = true
				}
				if v.Record && !v.recording {
					// open
					firstFrame := v.preRingBuffer.Pop()
					popped := v.preRingBuffer.PopAll()
					preFrames := *NewImageList()
					preFrames.Set(popped)
					v.openRecord(firstFrame)
					v.writeRecord(firstFrame)
					firstFrame.Cleanup()
					for preFrames.Len() > 0 {
						cur := preFrames.Pop()
						v.writeRecord(cur)
						cur.Cleanup()
					}
					v.recording = true
				} else if !v.Record && v.recording {
					// close
					v.closeRecord()
					v.recording = false
				}

				origImg := img.Original
				if origImg.IsValid() {
					if v.recording {
						// write
						v.writeRecord(origImg)
						v.VideoStats.AddAccepted()
					} else {
						// buffer
						oldest := v.preRingBuffer.Push(*origImg.Ref())
						if oldest.IsValid() {
							v.VideoStats.AddDropped()
						}
						oldest.Cleanup()
					}
				}
				img.Cleanup()
			case <-v.timeoutTick.C:
				if !v.activity {
					v.Record = false
				}
				v.activity = false
			case <-v.secTick.C:
				if !v.startTime.IsZero() && v.isRecordExpired(v.startTime) {
					v.Record = false
				}
			}
		}
	}()
}

// Send Image to write
func (v *VideoWriter) Send(img ProcessedImage) {
	v.streamChan.Send(img)
}

// Close notified by caller that input stream is done/closed
func (v *VideoWriter) Close() {
	v.streamChan.Close()
}

// Wait until done
func (v *VideoWriter) Wait() {
	v.streamChan.Wait()
	<-v.done
	v.VideoStats.Cleanup()
}

func (v *VideoWriter) openRecord(img Image) {
	timeNow := time.Now()
	saveFilenameFull := GetVideoFilename(timeNow, v.saveDirectory, v.name, v.fileType, false)
	wFull, err := gocv.VideoWriterFile(saveFilenameFull,
		strings.ToUpper(v.codec), float64(v.outFps),
		img.Width(), img.Height(), true)
	if err == nil {
		v.startTime = timeNow
		v.writerFull = wFull
		if v.savePreview {
			SavePreview(img, timeNow, v.saveDirectory, v.name, "", "")
		}
	} else {
		log.Error("Could not open gocv writer full")
	}
	if v.savePortable {
		scaledImage := img.ScaleToWidth(v.PortableWidth)
		saveFilenamePortable := GetVideoFilename(timeNow, v.saveDirectory, v.name, v.fileType, true)
		wPortable, err := gocv.VideoWriterFile(saveFilenamePortable,
			strings.ToUpper(v.codec), float64(v.outFps),
			scaledImage.Width(), scaledImage.Height(), true)
		if err == nil {
			v.writerPortable = wPortable
		} else {
			log.Error("Could not open gocv writer portable")
		}
		scaledImage.Cleanup()
	}
	return
}

func (v *VideoWriter) closeRecord() {
	if v.writerFull != nil {
		v.writerFull.Close()
	}
	if v.writerPortable != nil {
		v.writerPortable.Close()
	}
	v.startTime = time.Time{}
}

func cleanupRingBuffer(ringBuffer *RingBufferImage) {
	for ringBuffer.Len() > 0 {
		oldest := ringBuffer.Pop()
		oldest.Cleanup()
	}
}

func (v *VideoWriter) isRecordExpired(start time.Time) bool {
	return time.Now().Sub(start) > (time.Duration(v.maxSec) * time.Second)
}

func (v *VideoWriter) writeRecord(img Image) {
	if v.writerFull != nil {
		if img.SharedMat != nil {
			img.SharedMat.Guard.RLock()
			if sharedmat.Valid(&img.SharedMat.Mat) {
				v.writerFull.Write(img.SharedMat.Mat)
			}
			img.SharedMat.Guard.RUnlock()
		}
	}
	if v.writerPortable != nil {
		scaledImage := img.ScaleToWidth(v.PortableWidth)
		if scaledImage.SharedMat != nil {
			scaledImage.SharedMat.Guard.RLock()
			if sharedmat.Valid(&scaledImage.SharedMat.Mat) {
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

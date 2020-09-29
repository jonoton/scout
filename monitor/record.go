package monitor

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jonoton/scout/dir"
	"github.com/jonoton/scout/videosource"
)

// Record buffers ProcessedImages and writes to disk
type Record struct {
	name          string
	saveDirectory string
	RecordConf    *RecordConfig
	writer        *videosource.VideoWriter
	streamChan    chan videosource.ProcessedImage
	done          chan bool
	hourTick      *time.Ticker
}

// NewRecord creates a new Record
func NewRecord(name string, saveDirectory string, recordConf *RecordConfig, outFps int) *Record {
	if saveDirectory == "" || recordConf == nil {
		return nil
	}
	recordDir := filepath.Clean(saveDirectory+"/recordings") + string(filepath.Separator)
	os.MkdirAll(recordDir, os.ModePerm)

	a := &Record{
		name:          name,
		saveDirectory: recordDir,
		RecordConf:    recordConf,
		writer: videosource.NewVideoWriter(name, recordDir, "XVID", "mp4", recordConf.MaxPreSec,
			recordConf.TimeoutSec, recordConf.MaxSec, outFps, true, true, videosource.ActivityObject),
		streamChan: make(chan videosource.ProcessedImage),
		done:       make(chan bool),
		hourTick:   time.NewTicker(time.Hour),
	}
	return a
}

// Wait until done
func (r *Record) Wait() {
	<-r.done
}

// Start the processes
func (r *Record) Start() {
	go func() {
		defer close(r.done)
		r.writer.Start()
	Loop:
		for {
			select {
			case <-r.hourTick.C:
				r.prune()
			case img, ok := <-r.streamChan:
				if !ok {
					break Loop
				}
				r.process(img)
			}
		}
		r.prune()
		r.writer.Close()
		r.writer.Wait()
	}()
}

func (r *Record) process(img videosource.ProcessedImage) {
	if r.RecordConf.RecordObjects && len(img.Objects) > 0 {
		r.writer.Record = true
	}
	r.writer.Send(img)
	img.Cleanup()
}

func (r *Record) prune() {
	r.deleteOldRecordings()
	r.deleteWhenFull()
}

func (r *Record) deleteOldRecordings() {
	expiredFiles, _ := dir.Expired(r.saveDirectory, dir.RegexBeginsWith(r.name),
		time.Now(), time.Duration(r.RecordConf.DeleteAfterHours)*time.Hour)
	for _, fileInfo := range expiredFiles {
		fullPath := filepath.Clean(r.saveDirectory + string(filepath.Separator) + fileInfo.Name())
		err := os.Remove(fullPath)
		if err != nil {
			log.Errorln(err)
		}
	}
}

func (r *Record) deleteWhenFull() {
	dirSize, _ := dir.Size(r.saveDirectory, dir.RegexBeginsWith(r.name))
	if int(dir.BytesToGigaBytes(dirSize)) > r.RecordConf.DeleteAfterGB {
		files, _ := dir.List(r.saveDirectory, dir.RegexBeginsWith(r.name))
		sort.Sort(dir.DescendingTime(files))
		for _, fileInfo := range files {
			if int(dir.BytesToGigaBytes(dirSize)) <= r.RecordConf.DeleteAfterGB {
				break
			}
			dirSize -= uint64(fileInfo.Size())
			fullPath := filepath.Clean(r.saveDirectory + string(filepath.Separator) + fileInfo.Name())
			err := os.Remove(fullPath)
			if err != nil {
				log.Errorln(err)
			}
		}
	}
}

// Send a processed image to buffer
func (r *Record) Send(img videosource.ProcessedImage) {
	r.streamChan <- *img.Clone()
}

// Close notified by caller that input stream is done/closed
func (r *Record) Close() {
	close(r.streamChan)
}

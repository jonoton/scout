package monitor

import (
	"math"
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
	codec := "mp4v"
	if len(recordConf.Codec) == 4 {
		codec = recordConf.Codec
	}
	fileType := "mp4"
	if len(recordConf.FileType) >= 3 {
		fileType = recordConf.FileType
	}
	saveFull := !recordConf.PortableOnly
	a := &Record{
		name:          name,
		saveDirectory: recordDir,
		RecordConf:    recordConf,
		writer: videosource.NewVideoWriter(name, recordDir, codec, fileType, recordConf.BufferSeconds, recordConf.MaxPreSec,
			recordConf.TimeoutSec, recordConf.MaxSec, outFps, true, true, saveFull, videosource.ActivityObject),
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
		r.writer.Start()
	Loop:
		for {
			select {
			case <-r.hourTick.C:
				r.prune()
			case img, ok := <-r.streamChan:
				if !ok {
					img.Cleanup()
					break Loop
				}
				r.process(img)
			}
		}
		r.hourTick.Stop()
		r.prune()
		r.writer.Close()
		r.writer.Wait()
		close(r.done)
	}()
}

func (r *Record) process(img videosource.ProcessedImage) {
	if r.RecordConf.RecordObjects && img.HasObject() {
		r.writer.Trigger()
	}
	r.writer.Send(img)
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
	if int(math.Ceil(dir.BytesToGigaBytes(dirSize))) > r.RecordConf.DeleteAfterGB {
		files, _ := dir.List(r.saveDirectory, dir.RegexBeginsWith(r.name))
		sort.Sort(dir.AscendingTime(files))
		for _, fileInfo := range files {
			if int(math.Ceil(dir.BytesToGigaBytes(dirSize))) <= r.RecordConf.DeleteAfterGB {
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
	r.streamChan <- img
}

// Close notified by caller that input stream is done/closed
func (r *Record) Close() {
	close(r.streamChan)
}

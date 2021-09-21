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

// Continuous buffers ProcessedImages and writes to disk
type Continuous struct {
	name           string
	saveDirectory  string
	ContinuousConf *ContinuousConfig
	writer         *videosource.VideoWriter
	streamChan     chan videosource.ProcessedImage
	done           chan bool
	hourTick       *time.Ticker
}

// NewContinuous creates a new Continuous
func NewContinuous(name string, saveDirectory string, continuousConf *ContinuousConfig, outFps int) *Continuous {
	if saveDirectory == "" || continuousConf == nil {
		return nil
	}
	continuousDir := filepath.Clean(saveDirectory+"/continuous") + string(filepath.Separator)
	os.MkdirAll(continuousDir, os.ModePerm)

	c := &Continuous{
		name:           name,
		saveDirectory:  continuousDir,
		ContinuousConf: continuousConf,
		writer: videosource.NewVideoWriter(name, continuousDir, "avc1", "mp4", 0,
			continuousConf.TimeoutSec, continuousConf.MaxSec, outFps, true, true, videosource.ActivityImage),
		streamChan: make(chan videosource.ProcessedImage),
		done:       make(chan bool),
		hourTick:   time.NewTicker(time.Hour),
	}
	return c
}

// Wait until done
func (c *Continuous) Wait() {
	<-c.done
}

// Start the processes
func (c *Continuous) Start() {
	go func() {
		c.writer.Start()
	Loop:
		for {
			select {
			case <-c.hourTick.C:
				c.prune()
			case img, ok := <-c.streamChan:
				if !ok {
					img.Cleanup()
					break Loop
				}
				c.process(img)
			}
		}
		c.hourTick.Stop()
		c.prune()
		c.writer.Close()
		c.writer.Wait()
		close(c.done)
	}()
}

func (c *Continuous) process(img videosource.ProcessedImage) {
	c.writer.Trigger()
	c.writer.Send(img)
}

func (c *Continuous) prune() {
	c.deleteOldContinuous()
	c.deleteWhenFull()
}

func (c *Continuous) deleteOldContinuous() {
	expiredFiles, _ := dir.Expired(c.saveDirectory, dir.RegexBeginsWith(c.name),
		time.Now(), time.Duration(c.ContinuousConf.DeleteAfterHours)*time.Hour)
	for _, fileInfo := range expiredFiles {
		fullPath := filepath.Clean(c.saveDirectory + string(filepath.Separator) + fileInfo.Name())
		err := os.Remove(fullPath)
		if err != nil {
			log.Errorln(err)
		}
	}
}

func (c *Continuous) deleteWhenFull() {
	dirSize, _ := dir.Size(c.saveDirectory, dir.RegexBeginsWith(c.name))
	if int(math.Ceil(dir.BytesToGigaBytes(dirSize))) > c.ContinuousConf.DeleteAfterGB {
		files, _ := dir.List(c.saveDirectory, dir.RegexBeginsWith(c.name))
		sort.Sort(dir.AscendingTime(files))
		for _, fileInfo := range files {
			if int(math.Ceil(dir.BytesToGigaBytes(dirSize))) <= c.ContinuousConf.DeleteAfterGB {
				break
			}
			dirSize -= uint64(fileInfo.Size())
			fullPath := filepath.Clean(c.saveDirectory + string(filepath.Separator) + fileInfo.Name())
			err := os.Remove(fullPath)
			if err != nil {
				log.Errorln(err)
			}
		}
	}
}

// Send a processed image to buffer
func (c *Continuous) Send(img videosource.ProcessedImage) {
	c.streamChan <- img
}

// Close notified by caller that input stream is done/closed
func (c *Continuous) Close() {
	close(c.streamChan)
}

package monitor

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jonoton/go-dir"
	pubsubmutex "github.com/jonoton/go-pubsubmutex"
	"github.com/jonoton/go-videosource"
)

const topicContinuousImages = "topic-continuous-images"

// Continuous buffers ProcessedImages and writes to disk
type Continuous struct {
	name           string
	saveDirectory  string
	ContinuousConf *ContinuousConfig
	writer         *videosource.VideoWriter
	pubsub         pubsubmutex.PubSub
	bufferSize     int
	done           chan bool
	cancel         chan bool
	cancelOnce     sync.Once
	hourTick       *time.Ticker
}

// NewContinuous creates a new Continuous
func NewContinuous(name string, saveDirectory string, continuousConf *ContinuousConfig, outFps int) *Continuous {
	if saveDirectory == "" || continuousConf == nil {
		return nil
	}
	continuousDir := filepath.Clean(saveDirectory+"/continuous") + string(filepath.Separator)
	os.MkdirAll(continuousDir, os.ModePerm)
	codec := "mp4v"
	if len(continuousConf.Codec) == 4 {
		codec = continuousConf.Codec
	}
	fileType := "mp4"
	if len(continuousConf.FileType) >= 3 {
		fileType = continuousConf.FileType
	}
	saveFull := !continuousConf.PortableOnly
	c := &Continuous{
		name:           name,
		saveDirectory:  continuousDir,
		ContinuousConf: continuousConf,
		writer: videosource.NewVideoWriter(name, continuousDir, codec, fileType, continuousConf.BufferSeconds, 0,
			continuousConf.TimeoutSec, continuousConf.MaxSec, outFps, true, true, saveFull, videosource.ActivityImage),
		pubsub:     *pubsubmutex.NewPubSub(),
		bufferSize: continuousConf.BufferSeconds * outFps,
		done:       make(chan bool),
		cancel:     make(chan bool),
		hourTick:   time.NewTicker(time.Hour),
	}
	pubsubmutex.RegisterTopic[*videosource.ProcessedImage](&c.pubsub, topicContinuousImages)

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
		imageSub, _ := pubsubmutex.Subscribe[*videosource.ProcessedImage](&c.pubsub,
			topicContinuousImages, c.pubsub.GetUniqueSubscriberID(), c.bufferSize)
	Loop:
		for {
			select {
			case <-c.hourTick.C:
				c.prune()
			case msg, ok := <-imageSub.Ch:
				if !ok {
					if msg.Data != nil {
						img := msg.Data
						img.Cleanup()
					}
					break Loop
				}
				if msg.Data == nil {
					continue
				}
				img := msg.Data
				c.process(*img)
			case <-c.cancel:
				break Loop
			}
		}
		imageSub.Unsubscribe()
		c.hourTick.Stop()
		c.prune()
		c.writer.Close()
		c.writer.Wait()
		c.pubsub.Close()
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
func (c *Continuous) Send(img *videosource.ProcessedImage) {
	pubsubmutex.Publish(&c.pubsub,
		pubsubmutex.Message[*videosource.ProcessedImage]{Topic: topicContinuousImages, Data: img})
}

// Close notified by caller that input stream is done/closed
func (c *Continuous) Close() {
	c.cancelOnce.Do(func() {
		close(c.cancel)
	})
}

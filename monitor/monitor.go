// monitor package

package monitor

import (
	"sync"
	"time"

	"github.com/jonoton/go-notify"

	pubsubmutex "github.com/jonoton/go-pubsubmutex"
	"github.com/jonoton/go-videosource"
	"github.com/jonoton/scout/face"
	"github.com/jonoton/scout/motion"
	"github.com/jonoton/scout/tensor"
	log "github.com/sirupsen/logrus"
)

const topicMonitorImages = "topic-monitor-images"
const topicGetMonitorFrameStats = "topic-get-monitor-frame-stats"
const topicCurrentMonitorFrameStats = "topic-current-monitor-frame-stats"

// Monitor contains the video source
type Monitor struct {
	Name            string
	bufferSize      int
	ConfigPaths     []string
	reader          *videosource.VideoReader
	record          *Record
	continuous      *Continuous
	notifier        *notify.Notify
	notifyRxConf    *notify.RxConfig
	staleTimeout    int
	StaleRetry      int
	StaleMaxRetry   int
	IsStale         bool
	frameStatsCombo videosource.FrameStatsCombo
	motion          *motion.Motion
	tensor          *tensor.Tensor
	face            *face.Face
	alert           *Alert
	pubsub          pubsubmutex.PubSub
	done            chan bool
}

// Map is a map of names to Monitor
type Map map[string]*Monitor

// NewMonitor creates a new Monitor
func NewMonitor(name string, reader *videosource.VideoReader) *Monitor {
	m := &Monitor{
		Name:            name,
		bufferSize:      0,
		reader:          reader,
		record:          nil,
		continuous:      nil,
		notifier:        nil,
		notifyRxConf:    nil,
		staleTimeout:    20,
		StaleRetry:      10,
		StaleMaxRetry:   10,
		IsStale:         false,
		frameStatsCombo: videosource.FrameStatsCombo{},
		motion:          motion.NewMotion(name),
		tensor:          tensor.NewTensor(name),
		face:            face.NewFace(name),
		pubsub:          *pubsubmutex.NewPubSub(),
		alert:           nil,
		done:            make(chan bool),
	}
	pubsubmutex.RegisterTopic[*videosource.ProcessedImage](&m.pubsub, topicMonitorImages)
	pubsubmutex.RegisterTopic[any](&m.pubsub, topicGetMonitorFrameStats)
	pubsubmutex.RegisterTopic[*videosource.FrameStatsCombo](&m.pubsub, topicCurrentMonitorFrameStats)

	return m
}

func (m *Monitor) SetBufferSeconds(sec int) {
	if sec > 0 {
		m.bufferSize = m.reader.MaxOutputFps * sec
	} else {
		m.bufferSize = 0
	}
}

// SetStaleConfig sets the stale configuration
func (m *Monitor) SetStaleConfig(timeout int, maxRetry int) {
	if timeout > 0 {
		m.staleTimeout = timeout
	}
	if maxRetry > 0 {
		m.StaleRetry = maxRetry
		m.StaleMaxRetry = maxRetry
	}
}

// SetRecord sets the recorder
func (m *Monitor) SetRecord(saveDirectory string, recordConf *RecordConfig) {
	m.record = NewRecord(m.Name, saveDirectory, recordConf, m.reader.MaxOutputFps)
}

// SetContinuous sets the continuous recording
func (m *Monitor) SetContinuous(saveDirectory string, continuousConf *ContinuousConfig) {
	m.continuous = NewContinuous(m.Name, saveDirectory, continuousConf, m.reader.MaxOutputFps)
}

// SetAlert sets the alert notification
func (m *Monitor) SetAlert(notifier *notify.Notify, notifyRxConf *notify.RxConfig, saveDirectory string, alertConf *AlertConfig) {
	m.alert = NewAlert(m.Name, notifier, notifyRxConf, saveDirectory, alertConf)
}

// SetMotion sets the Motion Config
func (m *Monitor) SetMotion(config *motion.Config) {
	m.motion.SetConfig(config)
}

// SetTensor sets the Tensor Config
func (m *Monitor) SetTensor(config *tensor.Config) {
	m.tensor.SetConfig(config)
}

// SetFace sets the Face Config
func (m *Monitor) SetFace(config *face.Config) {
	m.face.SetConfig(config)
}

// Start will run the processes
func (m *Monitor) Start() {
	go func() {
		readerOutput := m.reader.Start()

		motionInput := make(chan videosource.Image, m.bufferSize)
		motionOutput := m.motion.Run(motionInput)

		tensorInput := make(chan videosource.ProcessedImage, m.bufferSize)
		tensorOutput := m.tensor.Run(tensorInput)

		faceInput := make(chan videosource.ProcessedImage, m.bufferSize)
		faceOutput := m.face.Run(faceInput)

		wg := &sync.WaitGroup{}
		wg.Add(3)
		go motionToTensor(motionOutput, tensorInput, wg)
		go tensorToFace(tensorOutput, faceInput, wg)
		go m.processResults(faceOutput, wg)
		readerToMotion(readerOutput, motionInput)

		m.reader.Wait()
		wg.Wait()
		m.IsStale = true
		log.Infoln("Done monitor", m.Name)
		m.pubsub.Close()
		close(m.done)
	}()
}

func readerToMotion(inChan <-chan videosource.Image, outChan chan videosource.Image) {
	for img := range inChan {
		outChan <- img
	}
	close(outChan)
}

func motionToTensor(inChan <-chan videosource.ProcessedImage, outChan chan videosource.ProcessedImage, wg *sync.WaitGroup) {
	for img := range inChan {
		outChan <- img
	}
	close(outChan)
	wg.Done()
}

func tensorToFace(inChan <-chan videosource.ProcessedImage, outChan chan videosource.ProcessedImage, wg *sync.WaitGroup) {
	for img := range inChan {
		outChan <- img
	}
	close(outChan)
	wg.Done()
}

func (m *Monitor) processResults(inChan <-chan videosource.ProcessedImage, wg *sync.WaitGroup) {
	if m.continuous != nil {
		m.continuous.Start()
	}
	if m.record != nil {
		m.record.Start()
	}
	if m.alert != nil {
		m.alert.Start()
	}
	getMonFrameStatsSub, _ := pubsubmutex.Subscribe[any](&m.pubsub, topicGetMonitorFrameStats, m.pubsub.GetUniqueSubscriberID(), 10)
	defer getMonFrameStatsSub.Unsubscribe()
	sourceStatsSub := m.reader.GetSourceStatsSub()
	defer sourceStatsSub.Unsubscribe()
	outputStatsSub := m.reader.GetOutputStatsSub()
	defer outputStatsSub.Unsubscribe()
	staleTicker := time.NewTicker(time.Second)
	staleSec := 0
	lastTotal := 0
FaceLoop:
	for {
		select {
		case _, ok := <-getMonFrameStatsSub.Ch:
			if !ok {
				continue
			}
			m.pubMonitorFrameStats()
		case msg, ok := <-sourceStatsSub.Ch:
			if !ok || msg.Data == nil {
				continue
			}
			cur := msg.Data
			m.frameStatsCombo.In = *cur
		case msg, ok := <-outputStatsSub.Ch:
			if !ok || msg.Data == nil {
				continue
			}
			cur := msg.Data
			m.frameStatsCombo.Out = *cur
		case cur, ok := <-inChan:
			if !ok {
				cur.Cleanup()
				break FaceLoop
			}
			if m.alert != nil {
				m.alert.Push(cur.Ref())
			}
			if m.record != nil {
				m.record.Send(cur.Ref())
			}
			if m.continuous != nil {
				m.continuous.Send(cur.Ref())
			}
			pubsubmutex.Publish(&m.pubsub, pubsubmutex.Message[*videosource.ProcessedImage]{Topic: topicMonitorImages, Data: cur.Ref()})
			cur.Cleanup()
		case <-staleTicker.C:
			curTotal := m.frameStatsCombo.In.AcceptedTotal
			if lastTotal == curTotal {
				staleSec++
			} else {
				staleSec = 0
				m.IsStale = false
				m.StaleRetry = m.StaleMaxRetry
			}
			lastTotal = curTotal
			if staleSec >= m.staleTimeout {
				m.IsStale = true
			}
		}
	}
	staleTicker.Stop()
	if m.alert != nil {
		m.alert.Close()
		m.alert.Wait()
	}
	if m.record != nil {
		m.record.Close()
		m.record.Wait()
	}
	if m.continuous != nil {
		m.continuous.Close()
		m.continuous.Wait()
	}
	wg.Done()
}

// Stop will stop the processes
func (m *Monitor) Stop() {
	m.reader.Stop()
}

// Wait until done
func (m *Monitor) Wait() {
	<-m.done
}

// Subscribe to video images
func (m *Monitor) Subscribe(bufferSize int) (result *pubsubmutex.Subscriber[*videosource.ProcessedImage]) {
	r, err := pubsubmutex.Subscribe[*videosource.ProcessedImage](&m.pubsub, topicMonitorImages, m.pubsub.GetUniqueSubscriberID(), bufferSize)
	if err == nil && r != nil {
		result = r
	}
	return
}

// GetMonitorFrameStats returns the monitor's frame stats
func (m *Monitor) GetMonitorFrameStats(timeoutMs int) (result *videosource.FrameStatsCombo) {
	r, ok := pubsubmutex.SendReceive[any, *videosource.FrameStatsCombo](&m.pubsub,
		topicGetMonitorFrameStats, topicCurrentMonitorFrameStats,
		nil, timeoutMs)
	if ok && r != nil {
		result = r
	}
	return
}

func (m *Monitor) pubMonitorFrameStats() {
	pubsubmutex.Publish(&m.pubsub,
		pubsubmutex.Message[*videosource.FrameStatsCombo]{Topic: topicCurrentMonitorFrameStats, Data: &m.frameStatsCombo})
}

// GetAlertTimes returns the alert times
func (m *Monitor) GetAlertTimes() (result AlertTimes) {
	if m.alert != nil {
		return m.alert.LastAlert
	}
	return
}

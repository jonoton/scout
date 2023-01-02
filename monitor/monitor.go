// monitor package

package monitor

import (
	"sync"
	"time"

	"github.com/cskr/pubsub"
	"github.com/jonoton/scout/face"
	"github.com/jonoton/scout/motion"
	"github.com/jonoton/scout/notify"
	pubsubmutex "github.com/jonoton/scout/pubsubMutex"
	"github.com/jonoton/scout/tensor"
	"github.com/jonoton/scout/videosource"
	log "github.com/sirupsen/logrus"
)

const topicSubscribe = "topic-monitor-subscribe"
const topicUnsubscribe = "topic-monitor-unsubscribe"
const topicGetMonitorFrameStats = "topic-get-monitor-frame-stats"
const topicCurrentMonitorFrameStats = "topic-current-monitor-frame-stats"

type subscribeMonitor struct {
	key     string
	subChan chan videosource.ProcessedImage
}

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
	subscriptions   map[string]chan videosource.ProcessedImage
	alert           *Alert
	pubsub          pubsubmutex.PubSubMutex
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
		subscriptions:   make(map[string]chan videosource.ProcessedImage),
		pubsub:          *pubsubmutex.New(0),
		alert:           nil,
		done:            make(chan bool),
	}

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
		m.pubsub.Start()
		readerOutput := m.reader.Start()

		motionInput := make(chan videosource.Image, m.bufferSize)
		motionOutput := m.motion.Run(motionInput)

		tensorInput := make(chan videosource.ProcessedImage, m.bufferSize)
		tensorOutput := m.tensor.Run(tensorInput)

		faceInput := make(chan videosource.ProcessedImage, m.bufferSize)
		faceOutput := m.face.Run(faceInput)

		wg := &sync.WaitGroup{}
		wg.Add(3)
		// motion -> tensor
		go func() {
			for cur := range motionOutput {
				tensorInput <- cur
			}
			close(tensorInput)
			wg.Done()
		}()
		// tensor -> face
		go func() {
			for cur := range tensorOutput {
				faceInput <- cur
			}
			close(faceInput)
			wg.Done()
		}()
		// face -> process results
		go func() {
			if m.continuous != nil {
				m.continuous.Start()
			}
			if m.record != nil {
				m.record.Start()
			}
			if m.alert != nil {
				m.alert.Start()
			}
			subChan := m.pubsub.Sub(topicSubscribe)
			unsubChan := m.pubsub.Sub(topicUnsubscribe)
			getMonFrameStatsChan := m.pubsub.Sub(topicGetMonitorFrameStats)
			sourceStatsChan := m.reader.GetSourceStatsChan()
			outputStatsChan := m.reader.GetOutputStatsChan()
			staleTicker := time.NewTicker(time.Second)
			staleSec := 0
			lastTotal := 0
		FaceLoop:
			for {
				select {
				case _, ok := <-getMonFrameStatsChan:
					if !ok {
						continue
					}
					m.pubMonitorFrameStats()
				case cur, ok := <-sourceStatsChan:
					if !ok || cur == nil {
						continue
					}
					m.frameStatsCombo.In = *cur.(*videosource.FrameStats)
				case cur, ok := <-outputStatsChan:
					if !ok || cur == nil {
						continue
					}
					m.frameStatsCombo.Out = *cur.(*videosource.FrameStats)
				case subMon, ok := <-subChan:
					if !ok {
						continue
					}
					m.subscribe(subMon.(subscribeMonitor))
				case key, ok := <-unsubChan:
					if !ok {
						continue
					}
					m.unsubscribe(key.(string))
				case cur, ok := <-faceOutput:
					if !ok {
						cur.Cleanup()
						break FaceLoop
					}
					for _, val := range m.subscriptions {
						subImage := *cur.Ref()
						select {
						case val <- subImage:
						default:
							subImage.Cleanup()
						}
					}
					if m.alert != nil {
						m.alert.Push(*cur.Ref())
					}
					if m.record != nil {
						m.record.Send(*cur.Ref())
					}
					if m.continuous != nil {
						m.continuous.Send(*cur.Ref())
					}
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
				m.alert.Stop()
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
			m.clearSubscriptions()
			wg.Done()
		}()

		// reader -> motion
		for img := range readerOutput {
			motionInput <- img
		}
		close(motionInput)

		m.reader.Wait()
		wg.Wait()
		m.pubsub.Shutdown()
		m.IsStale = true
		close(m.done)
		log.Infoln("Done monitor", m.Name)
	}()
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
func (m *Monitor) Subscribe(key string) (result chan videosource.ProcessedImage) {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		subMon := subscribeMonitor{
			key:     key,
			subChan: make(chan videosource.ProcessedImage),
		}
		result = subMon.subChan
		instance.TryPub(subMon, topicSubscribe)
	})
	return
}

// Subscribe to video images with channel
func (m *Monitor) SubscribeWithChan(key string, subChan chan videosource.ProcessedImage) {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		subMon := subscribeMonitor{
			key:     key,
			subChan: subChan,
		}
		instance.TryPub(subMon, topicSubscribe)
	})
}

// Unsubscribe to video images
func (m *Monitor) Unsubscribe(key string) {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		instance.TryPub(key, topicUnsubscribe)
	})
}

func (m *Monitor) subscribe(subMon subscribeMonitor) {
	m.subscriptions[subMon.key] = subMon.subChan
}
func (m *Monitor) unsubscribe(key string) {
	if _, found := m.subscriptions[key]; found {
		close(m.subscriptions[key])
		delete(m.subscriptions, key)
	}
}
func (m *Monitor) clearSubscriptions() {
	for _, val := range m.subscriptions {
		close(val)
	}
	m.subscriptions = make(map[string]chan videosource.ProcessedImage)
}

// GetMonitorFrameStats returns the monitor's frame stats
func (m *Monitor) GetMonitorFrameStats(timeoutMs int) (result *videosource.FrameStatsCombo) {
	r := m.pubsub.SendReceive(topicGetMonitorFrameStats, topicCurrentMonitorFrameStats,
		nil, timeoutMs)
	if r != nil {
		result = r.(*videosource.FrameStatsCombo)
	}
	return
}

func (m *Monitor) pubMonitorFrameStats() {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		instance.TryPub(&m.frameStatsCombo, topicCurrentMonitorFrameStats)
	})
}

// GetAlertTimes returns the alert times
func (m *Monitor) GetAlertTimes() (result AlertTimes) {
	if m.alert != nil {
		return m.alert.LastAlert
	}
	return
}

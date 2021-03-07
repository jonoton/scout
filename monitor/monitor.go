// monitor package

package monitor

import (
	"sync"
	"time"

	"github.com/jonoton/scout/face"
	"github.com/jonoton/scout/motion"
	"github.com/jonoton/scout/notify"
	"github.com/jonoton/scout/tensor"
	"github.com/jonoton/scout/videosource"
)

// Monitor contains the video source
type Monitor struct {
	Name                string
	ConfigPaths         []string
	reader              *videosource.VideoReader
	record              *Record
	notifier            *notify.Notify
	notifyRxConf        *notify.RxConfig
	notifySaveDirectory string
	staleTimeout        int
	StaleRetry          int
	StaleMaxRetry       int
	IsStale             bool
	motion              *motion.Motion
	tensor              *tensor.Tensor
	face                *face.Face
	subscriptions       map[string]chan videosource.ProcessedImage
	subGuard            sync.RWMutex
	alert               *Alert
	recordObjects       bool
	notifyObjects       bool
	sentEmail           int
	done                chan bool
}

// Map is a map of names to Monitor
type Map map[string]*Monitor

// NewMonitor creates a new Monitor
func NewMonitor(name string, reader *videosource.VideoReader) *Monitor {
	m := &Monitor{
		Name:                name,
		reader:              reader,
		record:              nil,
		notifier:            nil,
		notifyRxConf:        nil,
		notifySaveDirectory: "",
		staleTimeout:        20,
		StaleRetry:          10,
		StaleMaxRetry:       10,
		IsStale:             false,
		motion:              motion.NewMotion(),
		tensor:              tensor.NewTensor(),
		face:                face.NewFace(),
		subscriptions:       make(map[string]chan videosource.ProcessedImage, 0),
		subGuard:            sync.RWMutex{},
		alert:               nil,
		recordObjects:       false,
		notifyObjects:       false,
		sentEmail:           0,
		done:                make(chan bool),
	}

	return m
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
		staleTicker := time.NewTicker(time.Second)
		defer staleTicker.Stop()
		staleSec := 0
		lastTotal := 0
		for {
			select {
			case <-m.done:
				return
			case <-staleTicker.C:
				curTotal := m.reader.SourceStats.AcceptedTotal
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
					return
				}
			}
		}
	}()
	go func() {
		defer close(m.done)

		readerOutput := m.reader.Start()

		motionInput := make(chan videosource.Image)
		motionOutput := m.motion.Run(motionInput)

		tensorInput := make(chan videosource.ProcessedImage)
		tensorOutput := m.tensor.Run(tensorInput)

		faceInput := make(chan videosource.ProcessedImage)
		faceOutput := m.face.Run(faceInput)

		wg := sync.WaitGroup{}
		wg.Add(3)
		// motion -> tensor
		go func() {
			defer wg.Done()
			defer close(tensorInput)
			for cur := range motionOutput {
				tensorInput <- cur
			}
		}()
		// tensor -> face
		go func() {
			defer wg.Done()
			defer close(faceInput)
			for cur := range tensorOutput {
				faceInput <- cur
			}
		}()
		// face -> process results
		go func() {
			defer wg.Done()
			if m.record != nil {
				m.record.Start()
			}
			if m.alert != nil {
				m.alert.Start()
			}
			for cur := range faceOutput {
				m.subGuard.RLock()
				for _, val := range m.subscriptions {
					val <- *cur.Clone()
				}
				m.subGuard.RUnlock()

				if m.alert != nil {
					m.alert.Push(cur)
				}
				if m.record != nil {
					m.record.Send(cur)
				}
				cur.Cleanup()
			}
			if m.alert != nil {
				m.alert.Stop()
				m.alert.Wait()
			}
			if m.record != nil {
				m.record.Close()
				m.record.Wait()
			}
			m.clearSubscriptions()
		}()

		// reader -> motion
		for img := range readerOutput {
			motionInput <- img
		}
		close(motionInput)

		m.reader.Wait()
		wg.Wait()
		m.IsStale = true
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
func (m *Monitor) Subscribe(key string) <-chan videosource.ProcessedImage {
	m.subGuard.Lock()
	defer m.subGuard.Unlock()
	m.subscriptions[key] = make(chan videosource.ProcessedImage)
	return m.subscriptions[key]
}

// Unsubscribe to video images
func (m *Monitor) Unsubscribe(key string) {
	m.subGuard.Lock()
	defer m.subGuard.Unlock()
	if _, found := m.subscriptions[key]; found {
		close(m.subscriptions[key])
		delete(m.subscriptions, key)
	}
}

func (m *Monitor) clearSubscriptions() {
	m.subGuard.Lock()
	for _, val := range m.subscriptions {
		close(val)
	}
	m.subscriptions = make(map[string]chan videosource.ProcessedImage, 0)
	m.subGuard.Unlock()
}

// GetReaderInStats returns the reader source stats
func (m *Monitor) GetReaderInStats() *videosource.VideoStats {
	return m.reader.SourceStats
}

// GetReaderOutStats returns the reader output stats
func (m *Monitor) GetReaderOutStats() *videosource.VideoStats {
	return m.reader.OutputStats
}

// GetAlertTimes returns the alert times
func (m *Monitor) GetAlertTimes() (result AlertTimes) {
	if m.alert != nil {
		return m.alert.LastAlert
	}
	return
}

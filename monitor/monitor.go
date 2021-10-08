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
	Name             string
	bufferSize       int
	ConfigPaths      []string
	reader           *videosource.VideoReader
	record           *Record
	continuous       *Continuous
	notifier         *notify.Notify
	notifyRxConf     *notify.RxConfig
	staleTimeout     int
	StaleRetry       int
	StaleMaxRetry    int
	IsDoneProcessing bool
	IsStale          bool
	motion           *motion.Motion
	tensor           *tensor.Tensor
	face             *face.Face
	subscriptions    map[string]chan videosource.ProcessedImage
	subGuard         sync.RWMutex
	alert            *Alert
	done             chan bool
}

// Map is a map of names to Monitor
type Map map[string]*Monitor

// NewMonitor creates a new Monitor
func NewMonitor(name string, reader *videosource.VideoReader) *Monitor {
	m := &Monitor{
		Name:             name,
		bufferSize:       0,
		reader:           reader,
		record:           nil,
		continuous:       nil,
		notifier:         nil,
		notifyRxConf:     nil,
		staleTimeout:     20,
		StaleRetry:       10,
		StaleMaxRetry:    10,
		IsDoneProcessing: false,
		IsStale:          false,
		motion:           motion.NewMotion(),
		tensor:           tensor.NewTensor(),
		face:             face.NewFace(),
		subscriptions:    make(map[string]chan videosource.ProcessedImage),
		subGuard:         sync.RWMutex{},
		alert:            nil,
		done:             make(chan bool),
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
		staleTicker := time.NewTicker(time.Second)
		staleSec := 0
		lastTotal := 0
	Loop:
		for {
			select {
			case <-m.done:
				break Loop
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
					break Loop
				}
			}
		}
		staleTicker.Stop()
	}()
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
			for cur := range faceOutput {
				m.subGuard.RLock()
				for _, val := range m.subscriptions {
					select {
					case val <- *cur.Ref():
					default:
					}
				}
				m.subGuard.RUnlock()

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
			}
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
			m.IsDoneProcessing = true
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
		m.IsStale = true
		close(m.done)
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
func (m *Monitor) Subscribe(key string) (result <-chan videosource.ProcessedImage) {
	if m.IsDoneProcessing {
		return
	}

	m.subGuard.Lock()
	m.subscriptions[key] = make(chan videosource.ProcessedImage)
	result = m.subscriptions[key]
	m.subGuard.Unlock()
	return
}

// Unsubscribe to video images
func (m *Monitor) Unsubscribe(key string) {
	m.subGuard.Lock()
	if _, found := m.subscriptions[key]; found {
		close(m.subscriptions[key])
		delete(m.subscriptions, key)
	}
	m.subGuard.Unlock()
}

func (m *Monitor) clearSubscriptions() {
	m.subGuard.Lock()
	for _, val := range m.subscriptions {
		close(val)
	}
	m.subscriptions = make(map[string]chan videosource.ProcessedImage)
	m.subGuard.Unlock()
}

// GetReaderInStats returns the reader source stats
func (m *Monitor) GetReaderInStats() videosource.FrameStats {
	return m.reader.SourceStats.GetStats()
}

// GetReaderOutStats returns the reader output stats
func (m *Monitor) GetReaderOutStats() videosource.FrameStats {
	return m.reader.OutputStats.GetStats()
}

// GetAlertTimes returns the alert times
func (m *Monitor) GetAlertTimes() (result AlertTimes) {
	if m.alert != nil {
		return m.alert.LastAlert
	}
	return
}

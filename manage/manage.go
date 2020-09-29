// manage package

package manage

import (
	"sort"
	"sync"
	"time"

	"github.com/jonoton/scout/face"
	"github.com/jonoton/scout/motion"
	"github.com/jonoton/scout/tensor"

	log "github.com/sirupsen/logrus"

	"github.com/jonoton/scout/monitor"
	"github.com/jonoton/scout/notify"
	"github.com/jonoton/scout/runtime"
	"github.com/jonoton/scout/videosource"
)

// Manage contains all the monitors and manages them
type Manage struct {
	mons             monitor.Map
	monGuard         sync.RWMutex
	manageConf       Config
	notifySenderConf *notify.SenderConfig
	notifier         *notify.Notify
	done             chan bool
}

// NewManage creates a new Manage
func NewManage() *Manage {
	m := &Manage{
		mons:             make(monitor.Map),
		monGuard:         sync.RWMutex{},
		manageConf:       *NewConfig(runtime.GetRuntimeDirectory(".config") + ConfigFilename),
		notifySenderConf: notify.NewSenderConfig(runtime.GetRuntimeDirectory(".config") + notify.SenderConfigFilename),
		notifier:         nil,
		done:             make(chan bool),
	}
	if m.notifySenderConf != nil {
		m.notifier = notify.NewNotify(m.notifySenderConf.Host,
			m.notifySenderConf.Port,
			m.notifySenderConf.User,
			m.notifySenderConf.Password)
	}
	return m
}

// AddMonitor adds a new monitor to manage
func (m *Manage) AddMonitor(mon *monitor.Monitor) {
	m.monGuard.Lock()
	m.mons[mon.Name] = mon
	m.monGuard.Unlock()
	go func() {
		mon.Start()
		mon.Wait()
	}()
}

// GetMonitorNames returns a list of monitor names
func (m *Manage) GetMonitorNames() (result []string) {
	m.monGuard.RLock()
	defer m.monGuard.RUnlock()
	for key := range m.mons {
		result = append(result, key)
	}
	sort.Strings(result)
	return
}

// GetMonitorVideoStats returns the monitor's video stats
func (m *Manage) GetMonitorVideoStats(monitorName string) (readerIn *videosource.VideoStats, readerOut *videosource.VideoStats) {
	m.monGuard.RLock()
	defer m.monGuard.RUnlock()
	if mon, found := m.mons[monitorName]; found {
		readerIn = mon.GetReaderInStats()
		readerOut = mon.GetReaderOutStats()
	}
	return
}

// GetMonitorAlertTimes returns all monitor alert times
func (m *Manage) GetMonitorAlertTimes() (result map[string]monitor.AlertTimes) {
	m.monGuard.RLock()
	defer m.monGuard.RUnlock()
	result = make(map[string]monitor.AlertTimes, 0)
	for _, mon := range m.mons {
		result[mon.Name] = mon.GetAlertTimes()
	}
	return
}

// GetDataDirectory returns the save data directory
func (m *Manage) GetDataDirectory() string {
	return m.manageConf.Data
}

// Start runs the processes
func (m *Manage) Start() {
	for _, cur := range m.manageConf.Monitors {
		mon := m.setupMonitor(cur.Name, cur.ConfigPath)
		if mon != nil {
			m.AddMonitor(mon)
		}
	}
	m.checkStaleMonitors()
}

func (m *Manage) setupMonitor(name string, configPath string) (mon *monitor.Monitor) {
	if configPath == "" {
		return
	}
	monConf := monitor.NewConfig(runtime.GetRuntimeDirectory(".config") + configPath)
	if monConf == nil {
		log.Errorln("Could not setup", name)
		return
	}
	var video videosource.VideoSource
	if monConf.Filename != "" {
		video = videosource.NewFileSource(monConf.Filename, monConf.Filename)

	} else if monConf.URL != "" {
		video = videosource.NewIPCamSource(name, monConf.URL)
	} else {
		log.Errorln("No video source for", name)
		return
	}
	if video == nil {
		log.Errorln("Could not create video source for", name)
		return
	}
	videoReader := videosource.NewVideoReader(video, monConf.MaxSourceFps, monConf.MaxOutputFps)
	if videoReader == nil {
		log.Errorln("Could not create video reader for", name)
		return
	}
	videoReader.SetQuality(monConf.Quality)
	mon = monitor.NewMonitor(name, videoReader)
	if monConf.RecordFilename != "" {
		mon.SetRecord(m.manageConf.Data, monitor.NewRecordConfig(runtime.GetRuntimeDirectory(".config")+monConf.RecordFilename))
	}
	if monConf.AlertFilename != "" {
		alertSettings := monitor.NewAlertConfig(runtime.GetRuntimeDirectory(".config") + monConf.AlertFilename)
		if monConf.NotifyRxFilename != "" {
			notifyRxConf := notify.NewRxConfig(runtime.GetRuntimeDirectory(".config") + monConf.NotifyRxFilename)
			mon.SetAlert(m.notifier, notifyRxConf, m.manageConf.Data, alertSettings)
		} else {
			mon.SetAlert(m.notifier, nil, m.manageConf.Data, alertSettings)
		}
	}
	if monConf.MotionFilename != "" {
		mon.SetMotion(motion.NewConfig(runtime.GetRuntimeDirectory(".config") + monConf.MotionFilename))
	}
	if monConf.TensorFilename != "" {
		mon.SetTensor(tensor.NewConfig(runtime.GetRuntimeDirectory(".config") + monConf.TensorFilename))
	}
	if monConf.FaceFilename != "" {
		mon.SetFace(face.NewConfig(runtime.GetRuntimeDirectory(".config") + monConf.FaceFilename))
	}
	mon.SetStaleConfig(monConf.StaleTimeout, monConf.StaleMaxRetry)
	return mon
}

// Wait until done
func (m *Manage) Wait() {
	<-m.done
}

// Subscribe to a monitor's video images
func (m *Manage) Subscribe(monitorName string, key string) <-chan videosource.ProcessedImage {
	m.monGuard.RLock()
	defer m.monGuard.RUnlock()
	if mon, ok := m.mons[monitorName]; ok {
		return mon.Subscribe(key)
	}
	return nil
}

// Unsubscribe to a monitor's video images
func (m *Manage) Unsubscribe(monitorName string, key string) {
	m.monGuard.RLock()
	defer m.monGuard.RUnlock()
	if mon, ok := m.mons[monitorName]; ok {
		mon.Unsubscribe(key)
	}
}

func (m *Manage) checkStaleMonitors() {
	go func() {
		defer close(m.done)
		staleTicker := time.NewTicker(time.Second)
		defer staleTicker.Stop()
		lastStaleList := make([]*monitor.Monitor, 0)
		for {
			select {
			case <-staleTicker.C:
				staleList := make([]*monitor.Monitor, 0)
				m.monGuard.RLock()
				for _, cur := range m.mons {
					if cur.IsStale {
						staleList = append(staleList, cur)
						log.Warningln("Stale monitor", cur.Name)
					}
				}
				m.monGuard.RUnlock()
				for _, stale := range staleList {
					stale.Stop()
					stale.Wait()
					m.monGuard.Lock()
					delete(m.mons, stale.Name)
					m.monGuard.Unlock()
					if stale.StaleRetry == 0 {
						break
					}
					for _, conf := range m.manageConf.Monitors {
						if conf.Name == stale.Name {
							newMon := m.setupMonitor(conf.Name, conf.ConfigPath)
							for _, lastStale := range lastStaleList {
								if lastStale.Name == newMon.Name {
									newMon.StaleRetry = stale.StaleRetry - 1
									log.Warningln("Stale retry decremented monitor", newMon.Name)
									if newMon.StaleRetry == 0 {
										log.Errorln("Stale last retry for", newMon.Name)
									}
								}
							}
							m.AddMonitor(newMon)
							log.Infoln("Stale restarted monitor", newMon.Name)
							break
						}
					}
				}
				lastStaleList = staleList
			}
		}
	}()
}

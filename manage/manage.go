// manage package

package manage

import (
	"sort"
	"sync"
	"time"

	"github.com/jonoton/scout/face"
	"github.com/jonoton/scout/motion"
	"github.com/jonoton/scout/tensor"

	"github.com/radovskyb/watcher"
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
	wtr              *watcher.Watcher
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
		wtr:              watcher.New(),
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
	m.addMonitor(mon)
	m.monGuard.Unlock()
}
func (m *Manage) addMonitor(mon *monitor.Monitor) {
	m.mons[mon.Name] = mon
	for _, pathName := range mon.ConfigPaths {
		go func(pathName string) {
			m.wtr.Add(pathName)
		}(pathName)
	}
	mon.Start()
}

// GetMonitorNames returns a list of monitor names
func (m *Manage) GetMonitorNames() (result []string) {
	m.monGuard.RLock()
	for key := range m.mons {
		result = append(result, key)
	}
	m.monGuard.RUnlock()
	sort.Strings(result)
	return
}

// GetMonitorVideoStats returns the monitor's video stats
func (m *Manage) GetMonitorVideoStats(monitorName string) (readerIn videosource.FrameStats, readerOut videosource.FrameStats) {
	m.monGuard.RLock()
	if mon, found := m.mons[monitorName]; found {
		readerIn = mon.GetReaderInStats()
		readerOut = mon.GetReaderOutStats()
	}
	m.monGuard.RUnlock()
	return
}

// GetMonitorAlertTimes returns all monitor alert times
func (m *Manage) GetMonitorAlertTimes() (result map[string]monitor.AlertTimes) {
	m.monGuard.RLock()
	result = make(map[string]monitor.AlertTimes)
	for _, mon := range m.mons {
		result[mon.Name] = mon.GetAlertTimes()
	}
	m.monGuard.RUnlock()
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
	m.monitorConfigChanges()
}

func (m *Manage) setupMonitor(name string, configPath string) (mon *monitor.Monitor) {
	if configPath == "" {
		return
	}
	runtimeConfigDir := runtime.GetRuntimeDirectory(".config")
	monConfigPath := runtimeConfigDir + configPath
	monConf := monitor.NewConfig(monConfigPath)
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
	mon.ConfigPaths = append(mon.ConfigPaths, monConfigPath)
	if monConf.RecordFilename != "" {
		recordConfigPath := runtimeConfigDir + monConf.RecordFilename
		mon.SetRecord(m.manageConf.Data, monitor.NewRecordConfig(recordConfigPath))
		mon.ConfigPaths = append(mon.ConfigPaths, recordConfigPath)
	}
	if monConf.ContinuousFilename != "" {
		continuousConfigPath := runtimeConfigDir + monConf.ContinuousFilename
		mon.SetContinuous(m.manageConf.Data, monitor.NewContinuousConfig(continuousConfigPath))
		mon.ConfigPaths = append(mon.ConfigPaths, continuousConfigPath)
	}
	if monConf.AlertFilename != "" {
		alertPath := runtimeConfigDir + monConf.AlertFilename
		alertSettings := monitor.NewAlertConfig(alertPath)
		mon.ConfigPaths = append(mon.ConfigPaths, alertPath)
		if monConf.NotifyRxFilename != "" {
			notifyRxPath := runtimeConfigDir + monConf.NotifyRxFilename
			notifyRxConf := notify.NewRxConfig(notifyRxPath)
			mon.ConfigPaths = append(mon.ConfigPaths, notifyRxPath)
			mon.SetAlert(m.notifier, notifyRxConf, m.manageConf.Data, alertSettings)
		} else {
			mon.SetAlert(m.notifier, nil, m.manageConf.Data, alertSettings)
		}
	}
	if monConf.MotionFilename != "" {
		motionPath := runtimeConfigDir + monConf.MotionFilename
		mon.SetMotion(motion.NewConfig(motionPath))
		mon.ConfigPaths = append(mon.ConfigPaths, motionPath)
	}
	if monConf.TensorFilename != "" {
		tensorPath := runtimeConfigDir + monConf.TensorFilename
		mon.SetTensor(tensor.NewConfig(tensorPath))
		mon.ConfigPaths = append(mon.ConfigPaths, tensorPath)
	}
	if monConf.FaceFilename != "" {
		facePath := runtimeConfigDir + monConf.FaceFilename
		mon.SetFace(face.NewConfig(facePath))
		mon.ConfigPaths = append(mon.ConfigPaths, facePath)
	}
	mon.SetStaleConfig(monConf.StaleTimeout, monConf.StaleMaxRetry)
	mon.SetBufferSeconds(monConf.BufferSeconds)
	return mon
}

// Stop the manage
func (m *Manage) Stop() {
	m.monGuard.Lock()
	tmpMap := make(monitor.Map)
	for k, v := range m.mons {
		tmpMap[k] = v
	}
	for _, v := range tmpMap {
		m.removeMonitor(v)
	}
	m.monGuard.Unlock()
	close(m.done)
}

// Wait until done
func (m *Manage) Wait() {
	<-m.done
}

// Subscribe to a monitor's video images
func (m *Manage) Subscribe(monitorName string, key string) (result <-chan videosource.ProcessedImage) {
	m.monGuard.RLock()
	if mon, ok := m.mons[monitorName]; ok {
		result = mon.Subscribe(key)
	}
	m.monGuard.RUnlock()
	return
}

// Unsubscribe to a monitor's video images
func (m *Manage) Unsubscribe(monitorName string, key string) {
	m.monGuard.RLock()
	if mon, ok := m.mons[monitorName]; ok {
		mon.Unsubscribe(key)
	}
	m.monGuard.RUnlock()
}

func (m *Manage) doCheckStaleMonitors(lastStaleList []*monitor.Monitor) (staleList []*monitor.Monitor) {
	m.monGuard.Lock()
	staleList = make([]*monitor.Monitor, 0)
	for _, cur := range m.mons {
		if cur.IsStale {
			staleList = append(staleList, cur)
			log.Warningln("Stale monitor", cur.Name)
		}
	}
	for _, stale := range staleList {
		m.removeMonitor(stale)
		if stale.StaleRetry == 0 {
			continue
		}
		if found, conf := m.getMonitorConf(stale.Name); found {
			newMon := m.setupMonitor(conf.Name, conf.ConfigPath)
			if newMon == nil {
				continue
			}
			for _, lastStale := range lastStaleList {
				if lastStale.Name == newMon.Name {
					newMon.StaleRetry = stale.StaleRetry - 1
					log.Warningln("Stale retry decremented monitor", newMon.Name)
					if newMon.StaleRetry == 0 {
						log.Errorln("Stale last retry for", newMon.Name)
					}
				}
			}
			m.addMonitor(newMon)
			log.Infoln("Stale restarted monitor", newMon.Name)
		}
	}
	m.monGuard.Unlock()
	return
}

func (m *Manage) checkStaleMonitors() {
	go func() {
		staleTicker := time.NewTicker(time.Second)
		lastStaleList := make([]*monitor.Monitor, 0)
	Loop:
		for {
			select {
			case <-staleTicker.C:
				lastStaleList = m.doCheckStaleMonitors(lastStaleList)
			case <-m.done:
				break Loop
			}
		}
		staleTicker.Stop()
	}()
}

// RemoveMonitor will stop, wait, and remove from manage
func (m *Manage) RemoveMonitor(mon *monitor.Monitor) {
	m.monGuard.Lock()
	m.removeMonitor(mon)
	m.monGuard.Unlock()
}

func (m *Manage) removeMonitor(mon *monitor.Monitor) {
	mon.Stop()
	mon.Wait()
	uniquePaths := make(map[string]bool)
	for _, pathName := range mon.ConfigPaths {
		uniquePaths[pathName] = true
	}
	for _, cur := range m.mons {
		if cur == mon {
			continue
		}
		for _, pathName := range cur.ConfigPaths {
			if _, found := uniquePaths[pathName]; found {
				uniquePaths[pathName] = false
			}
		}
	}
	for pathName, unique := range uniquePaths {
		if unique {
			go func(pathName string) {
				m.wtr.Remove(pathName)
			}(pathName)
		}
	}
	delete(m.mons, mon.Name)
}

func (m *Manage) doMonitorConfigChanges(modPath string) {
	m.monGuard.Lock()
	log.Infoln("Config changed", modPath)
	aMons := m.associatedMonitors(modPath)
	for _, cur := range aMons {
		m.removeMonitor(cur)
		if found, conf := m.getMonitorConf(cur.Name); found {
			newMon := m.setupMonitor(conf.Name, conf.ConfigPath)
			if newMon == nil {
				continue
			}
			m.addMonitor(newMon)
			log.Infoln("Config restarted monitor", newMon.Name)
		}
	}
	m.monGuard.Unlock()
}

func (m *Manage) monitorConfigChanges() {
	go func() {
		for {
			select {
			case event := <-m.wtr.Event:
				m.doMonitorConfigChanges(event.Path)
			case <-m.wtr.Closed:
				return
			}
		}
	}()
	go func() {
		if err := m.wtr.Start(time.Millisecond * 500); err != nil {
			log.Errorln(err)
			return
		}
	}()
}

func (m *Manage) associatedMonitors(modPath string) (result []*monitor.Monitor) {
	result = make([]*monitor.Monitor, 0)
	for _, cur := range m.mons {
		for _, configPath := range cur.ConfigPaths {
			if configPath == modPath {
				result = append(result, cur)
				break
			}
		}
	}
	return
}

func (m *Manage) getMonitorConf(name string) (found bool, result mon) {
	for _, conf := range m.manageConf.Monitors {
		if conf.Name == name {
			found = true
			result = conf
			break
		}
	}
	return
}

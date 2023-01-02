// manage package

package manage

import (
	"sort"
	"time"

	"github.com/cskr/pubsub"
	"github.com/jonoton/scout/face"
	"github.com/jonoton/scout/motion"
	pubsubmutex "github.com/jonoton/scout/pubsubMutex"
	"github.com/jonoton/scout/tensor"

	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"

	"github.com/jonoton/scout/monitor"
	"github.com/jonoton/scout/notify"
	"github.com/jonoton/scout/runtime"
	"github.com/jonoton/scout/videosource"
)

const topicAddMon = "topic-add-mon"
const topicRemoveMon = "topic-remove-mon"
const topicSubscribe = "topic-manage-subscribe"
const topicUnsubscribe = "topic-manage-unsubscribe"
const topicStop = "topic-stop"
const topicGetMonitorNames = "topic-get-monitor-names"
const topicCurrentMonitorNames = "topic-current-monitor-names"
const topicGetMonitorFrameStats = "topic-get-monitor-frame-stats"
const topicCurrentMonitorFrameStats = "topic-current-monitor-frame-stats"
const topicGetMonitorAlertTimes = "topic-get-monitor-alert-times"
const topicCurrentMonitorAlertTimes = "topic-current-monitor-alert-times"

// Manage contains all the monitors and manages them
type Manage struct {
	mons             monitor.Map
	manageConf       Config
	notifySenderConf *notify.SenderConfig
	Notifier         *notify.Notify
	wtr              *watcher.Watcher
	pubsub           pubsubmutex.PubSubMutex
	done             chan bool
}

// NewManage creates a new Manage
func NewManage() *Manage {
	m := &Manage{
		mons:             make(monitor.Map),
		manageConf:       *NewConfig(runtime.GetRuntimeDirectory(".config") + ConfigFilename),
		notifySenderConf: notify.NewSenderConfig(runtime.GetRuntimeDirectory(".config") + notify.SenderConfigFilename),
		Notifier:         nil,
		wtr:              watcher.New(),
		pubsub:           *pubsubmutex.New(0),
		done:             make(chan bool),
	}
	if m.notifySenderConf != nil {
		m.Notifier = notify.NewNotify(m.notifySenderConf.Host,
			m.notifySenderConf.Port,
			m.notifySenderConf.User,
			m.notifySenderConf.Password)
	}
	return m
}

// AddMonitor adds a new monitor to manage
func (m *Manage) AddMonitor(mon *monitor.Monitor) {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		instance.TryPub(mon, topicAddMon)
	})
}
func (m *Manage) addMonitor(mon *monitor.Monitor) {
	log.Infoln("Add monitor", mon.Name)
	m.mons[mon.Name] = mon
	for _, pathName := range mon.ConfigPaths {
		go func(pathName string) {
			m.wtr.Add(pathName)
		}(pathName)
	}
	mon.Start()
}

// GetMonitorNames returns a list of monitor names
func (m *Manage) GetMonitorNames(timeoutMs int) (result []string) {
	r := m.pubsub.SendReceive(topicGetMonitorNames, topicCurrentMonitorNames,
		nil, timeoutMs)
	if r != nil {
		result = r.([]string)
	}
	return
}
func (m *Manage) pubMonitorNames() {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		result := make([]string, 0)
		for key := range m.mons {
			result = append(result, key)
		}
		sort.Strings(result)
		instance.TryPub(result, topicCurrentMonitorNames)
	})
}

// GetMonitorFrameStats returns the monitor's frame stats
func (m *Manage) GetMonitorFrameStats(monitorName string, timeoutMs int) (result *videosource.FrameStatsCombo) {
	r := m.pubsub.SendReceive(topicGetMonitorFrameStats, topicCurrentMonitorFrameStats,
		monitorName, timeoutMs)
	if r != nil {
		result = r.(*videosource.FrameStatsCombo)
	}
	return
}

func (m *Manage) pubMonitorFrameStats(monitorName string) {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		if mon, found := m.mons[monitorName]; found {
			combo := mon.GetMonitorFrameStats(200)
			instance.TryPub(combo, topicCurrentMonitorFrameStats)
		} else {
			instance.TryPub(nil, topicCurrentMonitorFrameStats)
		}
	})
}

// GetMonitorAlertTimes returns all monitor alert times
func (m *Manage) GetMonitorAlertTimes(timeoutMs int) (result map[string]monitor.AlertTimes) {
	r := m.pubsub.SendReceive(topicGetMonitorAlertTimes, topicCurrentMonitorAlertTimes,
		nil, timeoutMs)
	if r != nil {
		alertTimes := r.(map[string]monitor.AlertTimes)
		result = alertTimes
	}
	return
}
func (m *Manage) pubMonitorAlertTimes() {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		alertTimes := make(map[string]monitor.AlertTimes)
		for _, mon := range m.mons {
			alertTimes[mon.Name] = mon.GetAlertTimes()
		}
		instance.TryPub(alertTimes, topicCurrentMonitorAlertTimes)
	})
}

// GetDataDirectory returns the save data directory
func (m *Manage) GetDataDirectory() string {
	return m.manageConf.Data
}

// Start runs the processes
func (m *Manage) Start() {
	m.run()
}

func (m *Manage) addAllMonitors() {
	for _, cur := range m.manageConf.Monitors {
		mon := m.setupMonitor(cur.Name, cur.ConfigPath)
		if mon != nil {
			m.addMonitor(mon)
		}
	}
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
			mon.SetAlert(m.Notifier, notifyRxConf, m.manageConf.Data, alertSettings)
		} else {
			mon.SetAlert(m.Notifier, nil, m.manageConf.Data, alertSettings)
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
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		instance.TryPub(nil, topicStop)
	})
}
func (m *Manage) stop() {
	m.pubsub.Shutdown()
	tmpMap := make(monitor.Map)
	for k, v := range m.mons {
		tmpMap[k] = v
	}
	for _, v := range tmpMap {
		m.removeMonitor(v, true)
	}
	close(m.done)
}

// Wait until done
func (m *Manage) Wait() {
	<-m.done
}

type subscribeMonitor struct {
	monitorName string
	key         string
	subChan     chan videosource.ProcessedImage
}

// Subscribe to a monitor's video images
func (m *Manage) Subscribe(monitorName string, key string) (result <-chan videosource.ProcessedImage) {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		subMon := subscribeMonitor{
			monitorName: monitorName,
			key:         key,
			subChan:     make(chan videosource.ProcessedImage),
		}
		instance.TryPub(subMon, topicSubscribe)
		result = subMon.subChan
	})
	return
}
func (m *Manage) subscribe(subMon subscribeMonitor) {
	if mon, ok := m.mons[subMon.monitorName]; ok {
		mon.SubscribeWithChan(subMon.key, subMon.subChan)
	} else {
		close(subMon.subChan)
	}
}

// Unsubscribe to a monitor's video images
func (m *Manage) Unsubscribe(monitorName string, key string) {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		instance.TryPub(subscribeMonitor{
			monitorName: monitorName,
			key:         key,
			subChan:     nil,
		}, topicUnsubscribe)
	})
}
func (m *Manage) unsubscribe(subMon subscribeMonitor) {
	if mon, ok := m.mons[subMon.monitorName]; ok {
		mon.Unsubscribe(subMon.key)
	}
}

func (m *Manage) doCheckStaleMonitors(lastStaleList []*monitor.Monitor) (staleList []*monitor.Monitor) {
	staleList = make([]*monitor.Monitor, 0)
	for _, cur := range m.mons {
		if cur.IsStale {
			staleList = append(staleList, cur)
			log.Warningln("Stale monitor", cur.Name)
		}
	}
	for _, stale := range staleList {
		m.removeMonitor(stale, true)
		if stale.StaleRetry == 0 {
			log.Errorln("Stale monitor DONE retrying for", stale.Name)
			continue
		}
		if found, conf := m.getMonitorConf(stale.Name); found {
			newMon := m.setupMonitor(conf.Name, conf.ConfigPath)
			if newMon == nil {
				log.Errorln("Stale setup monitor FAILED for", stale.Name)
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
	return
}

func (m *Manage) run() {
	m.pubsub.Start()
	m.monitorConfigChanges()
	go func() {
		m.addAllMonitors()
		addMonChan := m.pubsub.Sub(topicAddMon)
		removeMonChan := m.pubsub.Sub(topicRemoveMon)
		subChan := m.pubsub.Sub(topicSubscribe)
		unsubChan := m.pubsub.Sub(topicUnsubscribe)
		stopChan := m.pubsub.Sub(topicStop)
		getMonNamesChan := m.pubsub.Sub(topicGetMonitorNames)
		getMonFrameStatsChan := m.pubsub.Sub(topicGetMonitorFrameStats)
		getMonAlertTimesChan := m.pubsub.Sub(topicGetMonitorAlertTimes)
		staleTicker := time.NewTicker(time.Second)
		lastStaleList := make([]*monitor.Monitor, 0)
		retryList := make([]mon, 0)
	Loop:
		for {
			select {
			case mon, ok := <-removeMonChan:
				if !ok {
					continue
				}
				m.removeMonitor(mon.(*monitor.Monitor), true)
			case mon, ok := <-addMonChan:
				if !ok {
					continue
				}
				m.addMonitor(mon.(*monitor.Monitor))
			case subMon, ok := <-subChan:
				if !ok {
					continue
				}
				m.subscribe(subMon.(subscribeMonitor))
			case subMon, ok := <-unsubChan:
				if !ok {
					continue
				}
				m.unsubscribe(subMon.(subscribeMonitor))
			case _, ok := <-stopChan:
				if !ok {
					continue
				}
				m.stop()
			case _, ok := <-getMonNamesChan:
				if !ok {
					continue
				}
				m.pubMonitorNames()
			case name, ok := <-getMonFrameStatsChan:
				if !ok {
					continue
				}
				m.pubMonitorFrameStats(name.(string))
			case _, ok := <-getMonAlertTimesChan:
				if !ok {
					continue
				}
				m.pubMonitorAlertTimes()
			case <-staleTicker.C:
				lastStaleList = m.doCheckStaleMonitors(lastStaleList)
			case event, ok := <-m.wtr.Event:
				if !ok {
					continue
				}
				retryList = m.doMonitorConfigChanges(event.Path, retryList)
			case <-m.done:
				break Loop
			}
		}
		staleTicker.Stop()
	}()
}

// RemoveMonitor will stop, wait, and remove from manage
func (m *Manage) RemoveMonitor(mon *monitor.Monitor) {
	m.pubsub.Use(func(instance *pubsub.PubSub) {
		instance.TryPub(mon, topicRemoveMon)
	})
}

func (m *Manage) removeMonitor(mon *monitor.Monitor, removeWatchPaths bool) {
	log.Infoln("Remove monitor", mon.Name)
	mon.Stop()
	mon.Wait()
	if removeWatchPaths {
		m.removeMonitorWatchPaths(mon)
	}
	delete(m.mons, mon.Name)
}
func (m *Manage) removeMonitorWatchPaths(mon *monitor.Monitor) {
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
}

func (m *Manage) doMonitorConfigChanges(modPath string, inList []mon) (retryList []mon) {
	log.Infoln("Config changed", modPath)
	aMons := m.associatedMonitors(modPath)
	tryList := inList
	for _, cur := range aMons {
		m.removeMonitor(cur, false)
		if found, conf := m.getMonitorConf(cur.Name); found {
			tryList = append(tryList, conf)
		}
	}
	for _, conf := range tryList {
		newMon := m.setupMonitor(conf.Name, conf.ConfigPath)
		if newMon == nil {
			log.Warningln("Config change setup monitor FAILED for", conf.Name)
			retryList = append(retryList, conf)
			continue
		}
		m.addMonitor(newMon)
		log.Infoln("Config restarted monitor", newMon.Name)
	}
	return
}

func (m *Manage) monitorConfigChanges() {
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

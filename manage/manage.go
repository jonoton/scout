// manage package

package manage

import (
	"sort"
	"sync"
	"time"

	pubsubmutex "github.com/jonoton/go-pubsubmutex"
	"github.com/jonoton/scout/face"
	"github.com/jonoton/scout/motion"
	"github.com/jonoton/scout/tensor"

	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"

	"github.com/jonoton/go-notify"
	"github.com/jonoton/go-runtime"
	"github.com/jonoton/go-videosource"
	"github.com/jonoton/scout/monitor"
)

const topicAddMon = "topic-add-mon"
const topicRemoveMon = "topic-remove-mon"
const topicGetMonitorSubscribe = "topic-manage-get-monitor-subscribe"
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
	pubsub           pubsubmutex.PubSub
	cancel           chan bool
	cancelOnce       sync.Once
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
		pubsub:           *pubsubmutex.NewPubSub(),
		cancel:           make(chan bool),
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
	m.pubsub.Publish(pubsubmutex.Message{Topic: topicAddMon, Data: mon})
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
	result := make([]string, 0)
	for key := range m.mons {
		result = append(result, key)
	}
	sort.Strings(result)
	m.pubsub.Publish(pubsubmutex.Message{Topic: topicCurrentMonitorNames, Data: result})
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
	if mon, found := m.mons[monitorName]; found {
		combo := mon.GetMonitorFrameStats(200)
		m.pubsub.Publish(pubsubmutex.Message{Topic: topicCurrentMonitorFrameStats, Data: combo})
	} else {
		m.pubsub.Publish(pubsubmutex.Message{Topic: topicCurrentMonitorFrameStats, Data: nil})
	}
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
	alertTimes := make(map[string]monitor.AlertTimes)
	for _, mon := range m.mons {
		alertTimes[mon.Name] = mon.GetAlertTimes()
	}
	m.pubsub.Publish(pubsubmutex.Message{Topic: topicCurrentMonitorAlertTimes, Data: alertTimes})
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
	m.pubsub.Publish(pubsubmutex.Message{Topic: topicStop, Data: nil})
}
func (m *Manage) stop() {
	m.cancelOnce.Do(func() {
		close(m.cancel)
	})
}

// Wait until done
func (m *Manage) Wait() {
	<-m.done
}

type subscribeMonitor struct {
	monitorName       string
	responseTopicName string
}

// Subscribe to a monitor's video images
func (m *Manage) Subscribe(monitorName string, timeoutMs int) (result *pubsubmutex.Subscriber) {
	subMon := subscribeMonitor{
		monitorName:       monitorName,
		responseTopicName: monitorName + m.pubsub.GetUniqueSubscriberID()}
	r := m.pubsub.SendReceive(topicGetMonitorSubscribe, subMon.responseTopicName,
		subMon, timeoutMs)
	if r != nil {
		result = r.(*pubsubmutex.Subscriber)
	}
	return
}
func (m *Manage) subscribe(subMon subscribeMonitor) (result *pubsubmutex.Subscriber) {
	if mon, ok := m.mons[subMon.monitorName]; ok {
		result = mon.Subscribe()
	}
	return
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
	m.monitorConfigChanges()
	go func() {
		defer m.pubsub.Close()
		defer close(m.done)
		defer m.cleanupAllMonitors()

		m.addAllMonitors()

		addMonSub := m.pubsub.Subscribe(topicAddMon, m.pubsub.GetUniqueSubscriberID(), 10)
		defer addMonSub.Unsubscribe()
		removeMonSub := m.pubsub.Subscribe(topicRemoveMon, m.pubsub.GetUniqueSubscriberID(), 10)
		defer removeMonSub.Unsubscribe()
		getMonitorSubscribeSub := m.pubsub.Subscribe(topicGetMonitorSubscribe, m.pubsub.GetUniqueSubscriberID(), 10)
		defer getMonitorSubscribeSub.Unsubscribe()
		stopSub := m.pubsub.Subscribe(topicStop, m.pubsub.GetUniqueSubscriberID(), 10)
		defer stopSub.Unsubscribe()
		getMonNamesSub := m.pubsub.Subscribe(topicGetMonitorNames, m.pubsub.GetUniqueSubscriberID(), 10)
		defer getMonNamesSub.Unsubscribe()
		getMonFrameStatsSub := m.pubsub.Subscribe(topicGetMonitorFrameStats, m.pubsub.GetUniqueSubscriberID(), 10)
		defer getMonFrameStatsSub.Unsubscribe()
		getMonAlertTimesSub := m.pubsub.Subscribe(topicGetMonitorAlertTimes, m.pubsub.GetUniqueSubscriberID(), 10)
		defer getMonAlertTimesSub.Unsubscribe()

		staleTicker := time.NewTicker(time.Second)
		defer staleTicker.Stop()
		lastStaleList := make([]*monitor.Monitor, 0)
		retryList := make([]mon, 0)
	Loop:
		for {
			select {
			case msg, ok := <-removeMonSub.Ch:
				if !ok {
					continue
				}
				mon := msg.Data.(*monitor.Monitor)
				m.removeMonitor(mon, true)
			case msg, ok := <-addMonSub.Ch:
				if !ok {
					continue
				}
				mon := msg.Data.(*monitor.Monitor)
				m.addMonitor(mon)
			case msg, ok := <-getMonitorSubscribeSub.Ch:
				if !ok {
					continue
				}
				subMon := msg.Data.(subscribeMonitor)
				sub := m.subscribe(subMon)
				m.pubsub.Publish(pubsubmutex.Message{Topic: subMon.responseTopicName, Data: sub})
			case _, ok := <-stopSub.Ch:
				if !ok {
					continue
				}
				m.stop()
			case _, ok := <-getMonNamesSub.Ch:
				if !ok {
					continue
				}
				m.pubMonitorNames()
			case msg, ok := <-getMonFrameStatsSub.Ch:
				if !ok {
					continue
				}
				name := msg.Data.(string)
				m.pubMonitorFrameStats(name)
			case _, ok := <-getMonAlertTimesSub.Ch:
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
			case <-m.cancel:
				break Loop
			}
		}
	}()
}

func (m *Manage) cleanupAllMonitors() {
	tmpMap := make(monitor.Map)
	for k, v := range m.mons {
		tmpMap[k] = v
	}
	for _, v := range tmpMap {
		m.removeMonitor(v, true)
	}
}

// RemoveMonitor will stop, wait, and remove from manage
func (m *Manage) RemoveMonitor(mon *monitor.Monitor) {
	m.pubsub.Publish(pubsubmutex.Message{Topic: topicRemoveMon, Data: mon})
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

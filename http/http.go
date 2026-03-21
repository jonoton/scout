// To install swag: go install github.com/swaggo/swag/cmd/swag@latest
// To regenerate swagger docs: $(go env GOPATH)/bin/swag init -o http/docs -g http/http.go --propertyStrategy pascalcase
// @title Scout API
// @version 1.0
// @description This is the API for Scout, a video monitoring system.
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Type "Bearer " followed by a space and JWT token.

package http

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/google/uuid"
	"github.com/jonoton/go-dir"
	"github.com/jonoton/go-memory"
	"github.com/jonoton/go-runtime"
	"github.com/jonoton/scout/manage"
	logrus "github.com/sirupsen/logrus"
	"github.com/valyala/bytebufferpool"

	"github.com/gofiber/swagger"
	_ "github.com/jonoton/scout/http/docs"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Http manages the http server
type Http struct {
	httpConfig          *Config
	fiber               *fiber.App
	manage              *manage.Manage
	loginLogger         *log.Logger
	accessLogger        *log.Logger
	linkClients         []*linkClient
	linkRetry           int
	loginNeeded         bool
	loginSigningKey     string
	twoFactorCheck      map[string]twoFactorAttempt
	twoFactorMu         sync.Mutex
	twoFactorTimeoutSec int
	secTick             *time.Ticker
	done                chan bool
}

// NewHttp returns a new Http
func NewHttp(manage *manage.Manage) *Http {
	cfgPath := runtime.GetRuntimeDirectory(".config") + ConfigFilename
	conf := NewConfig(cfgPath)
	if conf == nil {
		logrus.Warnf("Optional HTTP config file %s not found. Proceeding with defaults.", cfgPath)
	}

	h := &Http{
		httpConfig:          conf,
		fiber:               fiber.New(),
		manage:              manage,
		loginLogger:         &log.Logger{},
		accessLogger:        &log.Logger{},
		linkClients:         make([]*linkClient, 0),
		linkRetry:           2,
		loginNeeded:         false,
		loginSigningKey:     uuid.New().String(),
		twoFactorCheck:      make(map[string]twoFactorAttempt),
		twoFactorTimeoutSec: 60,
		secTick:             time.NewTicker(time.Second),
		done:                make(chan bool),
	}
	h.setup()
	return h
}

func (h *Http) setup() {
	logDir := runtime.GetRuntimeDirectory(".logs")
	h.loginNeeded = h.httpConfig != nil && len(h.httpConfig.Users) > 0

	if h.loginNeeded {
		logrus.Infof("Login logs saved to: %s", logDir+"logins")
		h.loginLogger.SetOutput(&lumberjack.Logger{
			Filename:   logDir + "logins",
			MaxSize:    1,
			MaxBackups: 5,
			MaxAge:     28,
			Compress:   false,
		})
		logrus.Infof("Access logs saved to: %s", logDir+"access")
		h.accessLogger.SetOutput(&lumberjack.Logger{
			Filename:   logDir + "access",
			MaxSize:    1,
			MaxBackups: 5,
			MaxAge:     28,
			Compress:   false,
		})
	}
	if h.httpConfig != nil && h.httpConfig.LoginSigningKey != "" {
		h.loginSigningKey = h.httpConfig.LoginSigningKey
	}
	if h.httpConfig != nil {
		for _, curLink := range h.httpConfig.Links {
			lc := newLinkClient(curLink.Name, curLink.Url, curLink.User, curLink.Password)
			h.linkClients = append(h.linkClients, lc)
		}
		if h.httpConfig.LinkRetry > 0 {
			h.linkRetry = h.httpConfig.LinkRetry
		}
	}
	if h.httpConfig != nil && h.httpConfig.TwoFactorTimeoutSec > 0 {
		h.twoFactorTimeoutSec = h.httpConfig.TwoFactorTimeoutSec
	}

	limitPerSecond := 100
	if h.httpConfig != nil && h.httpConfig.LimitPerSecond > 0 {
		limitPerSecond = h.httpConfig.LimitPerSecond
	}
	cfg := limiter.Config{
		Expiration: 1 * time.Second, // seconds
		Max:        limitPerSecond,  // requests
	}

	h.fiber.Use(limiter.New(cfg))

	h.fiber.Use(compress.New(compress.Config{Level: compress.LevelDefault}))

	h.fiber.Static("/", runtime.GetRuntimeDirectory("http")+"/public")

	if h.httpConfig != nil && h.httpConfig.EnableSwagger {
		h.fiber.Get("/swagger/*", swagger.HandlerDefault)
	}

	h.fiber.Get("/", h.indexHandler)

	if h.loginNeeded {
		loginLimit := 10
		if h.httpConfig != nil && h.httpConfig.LoginLimitPerSecond > 0 {
			loginLimit = h.httpConfig.LoginLimitPerSecond
		}
		loginLimiterCfg := limiter.Config{
			Expiration: 1 * time.Second,
			Max:        loginLimit,
		}
		h.fiber.Use("/login", limiter.New(loginLimiterCfg))
		h.fiber.Post("/login", h.loginHandler)
		go func() {
			for range h.secTick.C {
				// check expired two factors
				h.twoFactorMu.Lock()
				for k, v := range h.twoFactorCheck {
					delta := time.Since(v.time)
					if delta > time.Second*time.Duration(h.twoFactorTimeoutSec) {
						delete(h.twoFactorCheck, k)
					}
				}
				h.twoFactorMu.Unlock()
			}
		}()
	}

	h.fiber.Use("/live/:name", func(c *fiber.Ctx) error {
		monitorName := c.Params("name")
		c.Locals("monitorName", monitorName)
		width := c.Query("width")
		c.Locals("width", width)
		quality := c.Query("quality")
		c.Locals("jpegQuality", quality)
		token := c.Query("token")
		if token != "" {
			c.Request().Header.Add("Authorization", "Bearer "+token)
		}
		return c.Next()
	})

	if h.loginNeeded {
		h.fiber.Use(h.loginMiddleware())
	}

	h.fiber.Use("/live/:name", func(c *fiber.Ctx) error {
		localsMonName := c.Locals("monitorName")
		localsWidth := c.Locals("width")
		localsJpegQuality := c.Locals("jpegQuality")
		if localsMonName == nil || localsWidth == nil || localsJpegQuality == nil {
			return c.Next()
		}
		monitorName := localsMonName.(string)
		monitorList := h.manage.GetMonitorNames(2000)
		for _, cur := range monitorList {
			if cur == monitorName {
				return c.Next()
			}
		}
		for _, cur := range h.linkClients {
			for _, lmonName := range cur.monitorNames {
				if lmonName == monitorName {
					width, err := strconv.Atoi(localsWidth.(string))
					if err != nil {
						width = 0
					}
					jpegQuality, err := strconv.Atoi(localsJpegQuality.(string))
					if err != nil {
						jpegQuality = 0
					}
					return cur.forwardWebsocket(monitorName, width, jpegQuality)(c)
				}
			}
		}
		return c.Next()
	})

	h.fiber.Get("/live/:name", h.liveMonitor())

	h.fiber.Get("/heartbeat", h.heartbeatHandler)

	h.fiber.Use("/info/list", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/info/list", h.infoListHandler)

	h.fiber.Use("/info/:name", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/info/:name", h.infoNameHandler)

	h.fiber.Use("/alerts/latest", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/alerts/latest", h.alertsLatestHandler)

	h.fiber.Use("/alerts/list", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/alerts/list", h.alertsListHandler)

	h.fiber.Use("/alerts/files/:name", h.alertsFilesHandler)

	h.fiber.Static("/alerts/files",
		filepath.Clean(h.manage.GetDataDirectory()+"/alerts"),
		fiber.Static{
			Compress:  true,
			ByteRange: true,
			Browse:    false,
		},
	)

	h.fiber.Use("/recordings/list", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/recordings/list", h.recordingsListHandler)

	h.fiber.Use("/recordings/files/:name", h.recordingsFilesHandler)

	h.fiber.Static("/recordings/files",
		filepath.Clean(h.manage.GetDataDirectory()+"/recordings"),
		fiber.Static{
			Compress:  true,
			ByteRange: true,
			Browse:    false,
		},
	)

	h.fiber.Use("/continuous/list", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/continuous/list", h.continuousListHandler)

	h.fiber.Use("/continuous/files/:name", h.continuousFilesHandler)

	h.fiber.Static("/continuous/files",
		filepath.Clean(h.manage.GetDataDirectory()+"/continuous"),
		fiber.Static{
			Compress:  true,
			ByteRange: true,
			Browse:    false,
		},
	)

	h.fiber.Use("/memory", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/memory", h.memoryHandler)
}

// indexHandler serves the main dashboard
// @Summary Access Dashboard
// @Description Serve the index.html template for the Scout dashboard.
// @Tags UI
// @Produce html
// @Success 200 {string} string "HTML content"
// @Router / [get]
func (h *Http) indexHandler(c *fiber.Ctx) error {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	tmpl := template.Must(template.ParseFiles(runtime.GetRuntimeDirectory("http") + "templates/index.html"))
	tmpl.Execute(buf, nil)
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return c.Send(buf.Bytes())
}

// heartbeatHandler handles the heartbeat endpoint
// @Summary System Heartbeat
// @Description Check if the Scout server is responsive.
// @Tags System
// @Security ApiKeyAuth
// @Success 200 {string} string "OK"
// @Router /heartbeat [get]
func (h *Http) heartbeatHandler(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}

// infoListHandler returns a list of monitor names
// @Summary List Monitors
// @Description Get a list of all available monitor names, including those from linked Scout instances.
// @Tags Info
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} nameListResp "List of monitor names"
// @Router /info/list [get]
func (h *Http) infoListHandler(c *fiber.Ctx) error {
	monitorList := h.manage.GetMonitorNames(2000)
	for _, cur := range h.linkClients {
		curList := cur.getMonList(h.linkRetry)
		monitorList = append(monitorList, curList...)
	}
	data := nameListResp{
		NameList: monitorList,
	}
	return c.JSON(data)
}

// infoNameHandler returns information for a specific monitor
// @Summary Get Monitor Info
// @Description Get detailed information (FPS, etc.) for a specific monitor by name.
// @Tags Info
// @Produce json
// @Security ApiKeyAuth
// @Param name path string true "Monitor Name"
// @Success 200 {object} monInfoResp "Monitor information"
// @Failure 500 {string} string "Internal Server Error"
// @Router /info/{name} [get]
func (h *Http) infoNameHandler(c *fiber.Ctx) error {
	monitorName := c.Params("name")
	for _, cur := range h.linkClients {
		found, info := cur.getMonInfo(monitorName, h.linkRetry)
		if found {
			return c.JSON(info)
		}
	}
	data := monInfoResp{
		Name:         monitorName,
		ReaderInFps:  0,
		ReaderOutFps: 0,
	}
	frameStatsCombo := h.manage.GetMonitorFrameStats(monitorName, 1000)
	if frameStatsCombo == nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	data.ReaderInFps = frameStatsCombo.In.AcceptedPerSecond
	data.ReaderOutFps = frameStatsCombo.Out.AcceptedPerSecond
	return c.JSON(data)
}

// alertsLatestHandler returns the latest alert times for all monitors
// @Summary Get Latest Alerts
// @Description Get the latest alert timestamps for Object, Person, and Face detections across all monitors.
// @Tags Alerts
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]map[string]string "Latest alert times by monitor (RFC3339 format)"
// @Router /alerts/latest [get]
func (h *Http) alertsLatestHandler(c *fiber.Ctx) error {
	monAlertTimes := h.manage.GetMonitorAlertTimes(1000)
	// RFC RFC3339 time used
	data := make(map[string]map[string]string)
	for monName, monAlertTime := range monAlertTimes {
		curAlerts := make(map[string]string)
		if !monAlertTime.Object.IsZero() {
			curAlerts["Object"] = monAlertTime.Object.Format(time.RFC3339)
		}
		if !monAlertTime.Person.IsZero() {
			curAlerts["Person"] = monAlertTime.Person.Format(time.RFC3339)
		}
		if !monAlertTime.Face.IsZero() {
			curAlerts["Face"] = monAlertTime.Face.Format(time.RFC3339)
		}
		if len(curAlerts) > 0 {
			data[monName] = curAlerts
		}
	}
	for _, cur := range h.linkClients {
		linkResult := cur.getAlertsLatest(h.linkRetry)
		for k, v := range linkResult {
			data[k] = v
		}
	}
	return c.JSON(data)
}

// alertsListHandler returns a list of alert filenames
// @Summary List alerts
// @Description Get a list of alert JPG files, sorted descending by time.
// @Tags Alerts
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} string "List of alert filenames"
// @Router /alerts/list [get]
func (h *Http) alertsListHandler(c *fiber.Ctx) error {
	data := make([]string, 0)
	files, _ := dir.Expired(filepath.Clean(h.manage.GetDataDirectory()+"/alerts"),
		dir.RegexEndsWith(".jpg"), time.Now(), time.Duration(5)*time.Second)
	sort.Sort(dir.DescendingTime(files))
	for _, fileInfo := range files {
		data = append(data, fileInfo.Name())
	}
	needSort := false
	for _, cur := range h.linkClients {
		linkResult := cur.getAlertsList(h.linkRetry)
		if len(linkResult) > 0 {
			data = append(data, linkResult...)
			needSort = true
		}
	}
	if needSort {
		sort.Sort(dir.DescendingTimeName(data))
	}
	return c.JSON(data)
}

// alertsFilesHandler serves alert files
// @Summary Get alert file
// @Description Retrieve a specific alert JPG file.
// @Tags Alerts
// @Security ApiKeyAuth
// @Param name path string true "Filename"
// @Success 200 {file} file "Alert image"
// @Router /alerts/files/{name} [get]
func (h *Http) alertsFilesHandler(c *fiber.Ctx) error {
	paramFilename := c.Params("name")
	filename := filepath.Clean(h.manage.GetDataDirectory() + "/alerts/" + paramFilename)
	if fileExists(filename) {
		return c.Next()
	}
	for _, cur := range h.linkClients {
		found, linkResult := cur.getAlertsFile(paramFilename, h.linkRetry)
		if found {
			return c.Send(linkResult)
		}
	}
	return c.Next()
}

// recordingsListHandler returns a list of recording filenames
// @Summary List recordings
// @Description Get a list of motion recordings, sorted descending by time.
// @Tags Recordings
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} string "List of recording filenames"
// @Router /recordings/list [get]
func (h *Http) recordingsListHandler(c *fiber.Ctx) error {
	data := make([]string, 0)
	files, _ := dir.Expired(filepath.Clean(h.manage.GetDataDirectory()+"/recordings"),
		dir.RegexEndsWithBeforeExt("Portable"), time.Now(), time.Duration(5)*time.Second)
	sort.Sort(dir.DescendingTime(files))
	for _, fileInfo := range files {
		data = append(data, fileInfo.Name())
	}
	needSort := false
	for _, cur := range h.linkClients {
		linkResult := cur.getRecordingsList(h.linkRetry)
		if len(linkResult) > 0 {
			data = append(data, linkResult...)
			needSort = true
		}
	}
	if needSort {
		sort.Sort(dir.DescendingTimeName(data))
	}
	return c.JSON(data)
}

// recordingsFilesHandler serves recording files
// @Summary Get recording file
// @Description Retrieve a specific motion recording file.
// @Tags Recordings
// @Security ApiKeyAuth
// @Param name path string true "Filename"
// @Success 200 {file} file "Recording file"
// @Router /recordings/files/{name} [get]
func (h *Http) recordingsFilesHandler(c *fiber.Ctx) error {
	paramFilename := c.Params("name")
	filename := filepath.Clean(h.manage.GetDataDirectory() + "/recordings/" + paramFilename)
	if fileExists(filename) {
		return c.Next()
	}
	for _, cur := range h.linkClients {
		found, linkResult := cur.getRecordingsFile(paramFilename, h.linkRetry)
		if found {
			return c.Send(linkResult)
		}
	}
	return c.Next()
}

// continuousListHandler returns a list of continuous recording filenames
// @Summary List continuous recordings
// @Description Get a list of continuous recordings, sorted descending by time.
// @Tags Continuous
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} string "List of continuous filenames"
// @Router /continuous/list [get]
func (h *Http) continuousListHandler(c *fiber.Ctx) error {
	data := make([]string, 0)
	files, _ := dir.Expired(filepath.Clean(h.manage.GetDataDirectory()+"/continuous"),
		dir.RegexEndsWithBeforeExt("Portable"), time.Now(), time.Duration(5)*time.Second)
	sort.Sort(dir.DescendingTime(files))
	for _, fileInfo := range files {
		data = append(data, fileInfo.Name())
	}
	needSort := false
	for _, cur := range h.linkClients {
		linkResult := cur.getContinuousList(h.linkRetry)
		if len(linkResult) > 0 {
			data = append(data, linkResult...)
			needSort = true
		}
	}
	if needSort {
		sort.Sort(dir.DescendingTimeName(data))
	}
	return c.JSON(data)
}

// continuousFilesHandler serves continuous recording files
// @Summary Get continuous file
// @Description Retrieve a specific continuous recording file.
// @Tags Continuous
// @Security ApiKeyAuth
// @Param name path string true "Filename"
// @Success 200 {file} file "Continuous recording file"
// @Router /continuous/files/{name} [get]
func (h *Http) continuousFilesHandler(c *fiber.Ctx) error {
	paramFilename := c.Params("name")
	filename := filepath.Clean(h.manage.GetDataDirectory() + "/continuous/" + paramFilename)
	if fileExists(filename) {
		return c.Next()
	}
	for _, cur := range h.linkClients {
		found, linkResult := cur.getContinuousFile(paramFilename, h.linkRetry)
		if found {
			return c.Send(linkResult)
		}
	}
	return c.Next()
}

// memoryHandler returns memory usage information
// @Summary Get Memory Usage
// @Description Get current memory usage statistics for the application and system.
// @Tags System
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]int "Memory usage info in MB"
// @Router /memory [get]
func (h *Http) memoryHandler(c *fiber.Ctx) error {
	mem := memory.NewMemory()
	type info struct {
		HeapAllocatedMB int
		HeapTotalMB     int
		RAMAppMB        int
		RAMSystemMB     int
	}
	data := info{
		HeapAllocatedMB: int(memory.BytesToMegaBytes(mem.HeapAllocatedBytes)),
		HeapTotalMB:     int(memory.BytesToMegaBytes(mem.HeapTotalBytes)),
		RAMAppMB:        int(memory.BytesToMegaBytes(mem.RAMAppBytes)),
		RAMSystemMB:     int(memory.BytesToMegaBytes(mem.RAMSystemBytes)),
	}
	return c.JSON(data)
}

// Listen on port
func (h *Http) Listen() {
	go func() {
		port := ":8080"
		if h.httpConfig != nil && h.httpConfig.Port > 0 {
			portNum := h.httpConfig.Port
			port = fmt.Sprintf(":%d", portNum)
		}
		h.fiber.Listen(port)
	}()
}

func getFormattedKitchenTimestamp(t time.Time) string {
	return t.Format("03:04:05 PM 01-02-2006")
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func (h *Http) stopFiber() {
	stopTimeoutSec := 2
	done := make(chan bool)
	go func() {
		defer close(done)
		h.fiber.Shutdown()
	}()
	select {
	case <-done:
		logrus.Infoln("Stopped http fiber")
	case <-time.After(time.Duration(stopTimeoutSec) * time.Second):
		logrus.Infoln("Timeout waiting to stop http fiber")
	}
}

// Stop the http
func (h *Http) Stop() {
	defer close(h.done)
	h.secTick.Stop()
	h.stopFiber()
}

func (h *Http) Wait() {
	<-h.done
}

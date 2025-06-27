package http

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	twoFactorTimeoutSec int
	secTick             *time.Ticker
	done                chan bool
}

// NewHttp returns a new Http
func NewHttp(manage *manage.Manage) *Http {
	h := &Http{
		httpConfig:          NewConfig(runtime.GetRuntimeDirectory(".config") + ConfigFilename),
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
	h.loginLogger.SetOutput(&lumberjack.Logger{
		Filename:   logDir + "logins",
		MaxSize:    1,
		MaxBackups: 5,
		MaxAge:     28,
		Compress:   false,
	})
	h.accessLogger.SetOutput(&lumberjack.Logger{
		Filename:   logDir + "access",
		MaxSize:    1,
		MaxBackups: 5,
		MaxAge:     28,
		Compress:   false,
	})
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
	h.loginNeeded = h.httpConfig != nil && len(h.httpConfig.Users) > 0
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

	h.fiber.Get("/", func(c *fiber.Ctx) error {
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		tmpl := template.Must(template.ParseFiles(runtime.GetRuntimeDirectory("http") + "templates/index.html"))
		tmpl.Execute(buf, nil)
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
		return c.Send(buf.Bytes())
	})

	if h.loginNeeded {
		h.fiber.Post("/login", h.loginHandler)
		go func() {
			for range h.secTick.C {
				// check expired two factors
				for k, v := range h.twoFactorCheck {
					delta := time.Since(v.time)
					if delta > time.Second*time.Duration(h.twoFactorTimeoutSec) {
						delete(h.twoFactorCheck, k)
					}
				}
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
		c.Request().Header.Add("Authorization", "Bearer "+token)
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

	h.fiber.Get("/heartbeat", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	h.fiber.Use("/info/list", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/info/list", func(c *fiber.Ctx) error {
		monitorList := h.manage.GetMonitorNames(2000)
		for _, cur := range h.linkClients {
			curList := cur.getMonList(h.linkRetry)
			monitorList = append(monitorList, curList...)
		}
		data := nameListResp{
			NameList: monitorList,
		}
		return c.JSON(data)
	})

	h.fiber.Use("/info/:name", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/info/:name", func(c *fiber.Ctx) error {
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
	})

	h.fiber.Use("/alerts/latest", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/alerts/latest", func(c *fiber.Ctx) error {
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
	})

	h.fiber.Use("/alerts/list", cache.New(cache.Config{
		Expiration: 2 * time.Second,
	}))
	h.fiber.Get("/alerts/list", func(c *fiber.Ctx) error {
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
	})

	h.fiber.Use("/alerts/files/:name", func(c *fiber.Ctx) error {
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
	})

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
	h.fiber.Get("/recordings/list", func(c *fiber.Ctx) error {
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
	})

	h.fiber.Use("/recordings/files/:name", func(c *fiber.Ctx) error {
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
	})

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
	h.fiber.Get("/continuous/list", func(c *fiber.Ctx) error {
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
	})

	h.fiber.Use("/continuous/files/:name", func(c *fiber.Ctx) error {
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
	})

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
	h.fiber.Get("/memory", func(c *fiber.Ctx) error {
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
	})
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

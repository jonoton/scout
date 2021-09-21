package http

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/google/uuid"
	"github.com/jonoton/scout/dir"
	"github.com/jonoton/scout/manage"
	"github.com/jonoton/scout/memory"
	"github.com/jonoton/scout/runtime"
	"github.com/valyala/bytebufferpool"

	jwtware "github.com/gofiber/jwt/v2"
	"github.com/golang-jwt/jwt"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Http manages the http server
type Http struct {
	httpConfig   *Config
	fiber        *fiber.App
	manage       *manage.Manage
	loginLogger  *log.Logger
	accessLogger *log.Logger
	linkClients  []*linkClient
	linkRetry    int
}

// NewHttp returns a new Http
func NewHttp(manage *manage.Manage) *Http {
	h := &Http{
		httpConfig:   NewConfig(runtime.GetRuntimeDirectory(".config") + ConfigFilename),
		fiber:        fiber.New(),
		manage:       manage,
		loginLogger:  &log.Logger{},
		accessLogger: &log.Logger{},
		linkClients:  make([]*linkClient, 0),
		linkRetry:    2,
	}
	h.setup()
	return h
}

func getMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func (h *Http) validUser(user string, pass string) (bool, string) {
	if h.httpConfig == nil {
		return false, ""
	}
	for _, cur := range h.httpConfig.Users {
		if getMD5Hash(cur.User) == user && getMD5Hash(cur.Password) == pass {
			return true, cur.User
		}
	}
	return false, ""
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
	if h.httpConfig != nil {
		for _, curLink := range h.httpConfig.Links {
			lc := newLinkClient(curLink.Name, curLink.Url, curLink.User, curLink.Password)
			h.linkClients = append(h.linkClients, lc)
		}
		if h.httpConfig.LinkRetry > 0 {
			h.linkRetry = h.httpConfig.LinkRetry
		}
	}
	loginNeeded := h.httpConfig != nil && len(h.httpConfig.Users) > 0
	loginKey := uuid.New().String()
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

	if loginNeeded {
		h.fiber.Post("/login", func(c *fiber.Ctx) error {
			user := c.FormValue("a")
			pass := c.FormValue("b")
			timeNow := time.Now()
			if valid, vUser := h.validUser(user, pass); valid {
				// Create token
				token := jwt.New(jwt.SigningMethodHS256)

				// Set claims
				claims := token.Claims.(jwt.MapClaims)
				claims["user"] = user
				claims["exp"] = timeNow.Add(time.Hour * 24 * 7).Unix()
				if h.httpConfig != nil && h.httpConfig.SignInExpireDays > 0 {
					claims["exp"] = timeNow.Add(time.Hour * 24 * time.Duration(h.httpConfig.SignInExpireDays)).Unix()
				}

				// Generate encoded token and send it as response.
				t, err := token.SignedString([]byte(loginKey))
				if err != nil {
					h.loginLogger.Printf("%s,error,%s,%s,%s\r\n", getFormattedKitchenTimestamp(timeNow), vUser, c.IP(), c.IPs())
					return c.SendStatus(fiber.StatusInternalServerError)
				}
				h.loginLogger.Printf("%s,success,%s,%s,%s\r\n", getFormattedKitchenTimestamp(timeNow), vUser, c.IP(), c.IPs())
				return c.JSON(fiber.Map{"c": t})
			}
			h.loginLogger.Printf("%s,unauthorized,,%s,%s\r\n", getFormattedKitchenTimestamp(timeNow), c.IP(), c.IPs())
			return c.SendStatus(fiber.StatusUnauthorized)
		})
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

	if loginNeeded {
		h.fiber.Use(jwtware.New(jwtware.Config{
			SigningKey: []byte(loginKey),
			SuccessHandler: func(c *fiber.Ctx) error {
				h.accessLogger.Printf("%s,access,%s,%s\r\n", getFormattedKitchenTimestamp(time.Now()), c.IP(), c.IPs())
				return c.Next()
			},
			ErrorHandler: func(c *fiber.Ctx, e error) error {
				h.accessLogger.Printf("%s,%s,%s,%s\r\n", getFormattedKitchenTimestamp(time.Now()), e, c.IP(), c.IPs())
				return c.SendStatus(fiber.StatusUnauthorized)
			},
		}))
	}

	h.fiber.Use("/live/:name", func(c *fiber.Ctx) error {
		localsMonName := c.Locals("monitorName")
		localsWidth := c.Locals("width")
		localsJpegQuality := c.Locals("jpegQuality")
		if localsMonName == nil || localsWidth == nil || localsJpegQuality == nil {
			return c.Next()
		}
		monitorName := localsMonName.(string)
		monitorList := h.manage.GetMonitorNames()
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

	h.fiber.Get("/info/list", func(c *fiber.Ctx) error {
		monitorList := h.manage.GetMonitorNames()
		for _, cur := range h.linkClients {
			curList := cur.getMonList(h.linkRetry)
			monitorList = append(monitorList, curList...)
		}
		data := nameListResp{
			NameList: monitorList,
		}
		return c.JSON(data)
	})

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
		readerIn, readerOut := h.manage.GetMonitorVideoStats(monitorName)
		if readerIn != nil {
			data.ReaderInFps = readerIn.AcceptedPerSecond
		}
		if readerOut != nil {
			data.ReaderOutFps = readerOut.AcceptedPerSecond
		}
		return c.JSON(data)
	})

	h.fiber.Get("/alerts/latest", func(c *fiber.Ctx) error {
		monAlertTimes := h.manage.GetMonitorAlertTimes()
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

	h.fiber.Get("/recordings/list", func(c *fiber.Ctx) error {
		data := make([]string, 0)
		files, _ := dir.Expired(filepath.Clean(h.manage.GetDataDirectory()+"/recordings"),
			dir.RegexEndsWith("Full.mp4"), time.Now(), time.Duration(5)*time.Second)
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

	h.fiber.Get("/continuous/list", func(c *fiber.Ctx) error {
		data := make([]string, 0)
		files, _ := dir.Expired(filepath.Clean(h.manage.GetDataDirectory()+"/continuous"),
			dir.RegexEndsWith("Full.mp4"), time.Now(), time.Duration(5)*time.Second)
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

// Stop the http
func (h *Http) Stop() {
	h.fiber.Shutdown()
}

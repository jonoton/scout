package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	fiber "github.com/gofiber/fiber/v2"
	websocket "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/jonoton/scout/http/websockets"
	log "github.com/sirupsen/logrus"
)

const linkClientSeparator = "-"

type linkClient struct {
	name         string
	url          string
	user         string
	passHash     string
	token        string
	monitorNames []string
}

func newLinkClient(name string, url string, user string, password string) *linkClient {
	l := &linkClient{
		name:         name,
		url:          url,
		user:         user,
		passHash:     getSHA256Hash(password),
		token:        "",
		monitorNames: make([]string, 0),
	}
	return l
}

func (l *linkClient) trimName(s string) string {
	if l.name != "" {
		return strings.TrimPrefix(s, l.name+linkClientSeparator)
	}
	return s
}
func (l *linkClient) prependName(s string) string {
	if l.name != "" {
		return l.name + linkClientSeparator + s
	}
	return s
}

func (l *linkClient) login() {
	if l.user == "" || l.passHash == "" {
		return
	}
	l.token = ""
	agent := fiber.Post(l.url + "/login").InsecureSkipVerify()
	args := fiber.AcquireArgs()
	args.Set("a", getSHA256Hash(l.user))
	args.Set("b", l.passHash)
	agent.Form(args)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		if code == fiber.StatusOK {
			var r fiber.Map
			json.Unmarshal(body, &r)
			if c, ok := r["c"]; ok {
				l.token = fmt.Sprintf("%v", c)
			}
		}
	}
	fiber.ReleaseArgs(args)
}

func (l *linkClient) checkNeedLogin() {
	if l.user != "" && l.passHash != "" && l.token == "" {
		l.login()
	}
}

func (l *linkClient) checkClearLogin(code int) {
	if code == fiber.StatusForbidden || code == fiber.StatusUnauthorized {
		l.token = ""
	}
}

func (l *linkClient) checkAddAuth(agent *fiber.Agent) {
	if l.token != "" {
		agent.Set("Authorization", "Bearer "+l.token)
	}
}

type nameListResp struct {
	NameList []string
}

func (l *linkClient) getMonList(numRetries int) []string {
	l.checkNeedLogin()
	l.monitorNames = make([]string, 0)
	agent := fiber.Get(l.url + "/info/list").InsecureSkipVerify()
	l.checkAddAuth(agent)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		l.checkClearLogin(code)
		if code == fiber.StatusOK {
			var r nameListResp
			json.Unmarshal(body, &r)
			for _, cur := range r.NameList {
				monName := l.prependName(cur)
				l.monitorNames = append(l.monitorNames, monName)
			}
		} else if numRetries > 0 {
			numRetries--
			return l.getMonList(numRetries)
		}
	}
	return l.monitorNames
}

type monInfoResp struct {
	Name         string
	ReaderInFps  int
	ReaderOutFps int
}

func (l *linkClient) getMonInfo(name string, numRetries int) (found bool, result monInfoResp) {
	l.checkNeedLogin()
	var monName string
	for _, cur := range l.monitorNames {
		if cur == name {
			found = true
			monName = l.trimName(cur)
			break
		}
	}
	if !found {
		return
	}
	agent := fiber.Get(l.url + "/info/" + monName).InsecureSkipVerify()
	l.checkAddAuth(agent)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		l.checkClearLogin(code)
		if code == fiber.StatusOK {
			json.Unmarshal(body, &result)
			result.Name = l.prependName(result.Name)
		} else if numRetries > 0 {
			numRetries--
			return l.getMonInfo(name, numRetries)
		}
	}
	return
}

func (l *linkClient) getAlertsLatest(numRetries int) map[string]map[string]string {
	l.checkNeedLogin()
	result := make(map[string]map[string]string)
	agent := fiber.Get(l.url + "/alerts/latest").InsecureSkipVerify()
	l.checkAddAuth(agent)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		l.checkClearLogin(code)
		if code == fiber.StatusOK {
			var r map[string]map[string]string
			json.Unmarshal(body, &r)
			for k, v := range r {
				monName := l.prependName(k)
				result[monName] = v
			}
		} else if numRetries > 0 {
			numRetries--
			return l.getAlertsLatest(numRetries)
		}
	}
	return result
}

func (l *linkClient) getAlertsList(numRetries int) []string {
	l.checkNeedLogin()
	result := make([]string, 0)
	agent := fiber.Get(l.url + "/alerts/list").InsecureSkipVerify()
	l.checkAddAuth(agent)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		l.checkClearLogin(code)
		if code == fiber.StatusOK {
			json.Unmarshal(body, &result)
		} else if numRetries > 0 {
			numRetries--
			return l.getAlertsList(numRetries)
		}
	}
	return result
}

func (l *linkClient) getAlertsFile(filename string, numRetries int) (found bool, result []byte) {
	l.checkNeedLogin()
	result = make([]byte, 0)
	agent := fiber.Get(l.url + "/alerts/files/" + filename).InsecureSkipVerify()
	l.checkAddAuth(agent)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		l.checkClearLogin(code)
		if code == fiber.StatusOK {
			result = body
			found = true
		} else if numRetries > 0 {
			numRetries--
			return l.getAlertsFile(filename, numRetries)
		}
	}
	return
}

func (l *linkClient) getRecordingsList(numRetries int) []string {
	l.checkNeedLogin()
	result := make([]string, 0)
	agent := fiber.Get(l.url + "/recordings/list").InsecureSkipVerify()
	l.checkAddAuth(agent)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		l.checkClearLogin(code)
		if code == fiber.StatusOK {
			json.Unmarshal(body, &result)
		} else if numRetries > 0 {
			numRetries--
			return l.getRecordingsList(numRetries)
		}
	}
	return result
}

func (l *linkClient) getRecordingsFile(filename string, numRetries int) (found bool, result []byte) {
	l.checkNeedLogin()
	result = make([]byte, 0)
	agent := fiber.Get(l.url + "/recordings/files/" + filename).InsecureSkipVerify()
	l.checkAddAuth(agent)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		if code == fiber.StatusOK {
			result = body
			found = true
		} else if numRetries > 0 {
			numRetries--
			l.checkClearLogin(code)
			return l.getRecordingsFile(filename, numRetries)
		}
	}
	return
}

func (l *linkClient) getContinuousList(numRetries int) []string {
	l.checkNeedLogin()
	result := make([]string, 0)
	agent := fiber.Get(l.url + "/continuous/list").InsecureSkipVerify()
	l.checkAddAuth(agent)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		l.checkClearLogin(code)
		if code == fiber.StatusOK {
			json.Unmarshal(body, &result)
		} else if numRetries > 0 {
			numRetries--
			return l.getContinuousList(numRetries)
		}
	}
	return result
}

func (l *linkClient) getContinuousFile(filename string, numRetries int) (found bool, result []byte) {
	l.checkNeedLogin()
	result = make([]byte, 0)
	agent := fiber.Get(l.url + "/continuous/files/" + filename).InsecureSkipVerify()
	l.checkAddAuth(agent)
	if err := agent.Parse(); err == nil {
		code, body, _ := agent.Bytes()
		if code == fiber.StatusOK {
			result = body
			found = true
		} else if numRetries > 0 {
			numRetries--
			l.checkClearLogin(code)
			return l.getContinuousFile(filename, numRetries)
		}
	}
	return
}

func (l *linkClient) forwardWebsocket(monName string, width int, jpegQuality int) func(*fiber.Ctx) error {
	l.checkNeedLogin()
	sockMonName := l.trimName(monName)
	rawUrl := l.url + "/live/" + sockMonName
	queryArgs := make([]string, 0)
	if width > 0 {
		queryArgs = append(queryArgs, fmt.Sprintf("width=%d", width))
	}
	if jpegQuality > 0 {
		queryArgs = append(queryArgs, fmt.Sprintf("quality=%d", jpegQuality))
	}
	if len(l.token) > 0 {
		queryArgs = append(queryArgs, fmt.Sprintf("token=%s", l.token))
	}
	for index, curArg := range queryArgs {
		if index == 0 {
			rawUrl = rawUrl + "?"
		}
		if index > 0 {
			rawUrl = rawUrl + "&"
		}
		rawUrl = rawUrl + curArg
	}
	u, _ := url.Parse(rawUrl)
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	return websocket.New(func(c *websocket.Conn) {
		socketClosed := make(chan bool)
		dialer := gorillaWebsocket.DefaultDialer
		connBackend, _, err := dialer.Dial(u.String(), http.Header{})
		if err != nil {
			log.Warnln("Link Websocket connect error", u.Scheme, monName)
			return
		}
		uuid := uuid.New().String()
		log.Infoln("Link Websocket opened", u.Scheme, uuid)
		receive := func(msgType int, data []byte) {
			connBackend.WriteMessage(msgType, data)
		}
		send := func(c *websocket.Conn) {
		Loop:
			for {
				select {
				case <-socketClosed:
					break Loop
				default:
				}
				msgType, msg, err := connBackend.ReadMessage()
				if err != nil {
					m := websocket.FormatCloseMessage(websocket.CloseNormalClosure, fmt.Sprintf("%v", err))
					c.WriteMessage(websocket.CloseMessage, m)
					break Loop
				}
				err = c.WriteMessage(msgType, msg)
				if err != nil {
					break Loop
				}
			}
		}
		cleanup := func() {
			connBackend.Close()
			log.Infoln("Link Websocket closed", u.Scheme, uuid)
		}
		websockets.Run(c, socketClosed, receive, send, cleanup)
	})
}

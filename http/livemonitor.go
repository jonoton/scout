package http

import (
	"context"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	fiber "github.com/gofiber/fiber/v2"
	websocket "github.com/gofiber/websocket/v2"
	"github.com/jonoton/go-gzip"
	"github.com/jonoton/go-ringbuffer"
	"github.com/jonoton/go-videosource"
	"github.com/jonoton/go-websockets"
)

func (h *Http) liveMonitor() func(*fiber.Ctx) error {
	return websocket.New(func(c *websocket.Conn) {
		localsMonName := c.Locals("monitorName")
		if localsMonName == nil {
			log.Errorln("No monitor name")
			return
		}
		localsWidth := c.Locals("width")
		if localsWidth == nil {
			log.Errorln("No width")
			return
		}
		localsJpegQuality := c.Locals("jpegQuality")
		if localsJpegQuality == nil {
			log.Errorln("No jpeg quality")
			return
		}

		monitorName := localsMonName.(string)

		width, err := strconv.Atoi(localsWidth.(string))
		if err != nil {
			width = 0
		}
		jpegQuality, err := strconv.Atoi(localsJpegQuality.(string))
		if err != nil {
			jpegQuality = 60
		}

		imagesSub := h.manage.Subscribe(monitorName, 500)
		if imagesSub == nil {
			log.Errorln("Failed to subscribe to monitor", monitorName)
			return
		}

		websocketName := monitorName + "-" + imagesSub.ID
		log.Infoln("Websocket opened", websocketName)
		socketCtx, socketCancel := context.WithCancel(context.Background())
		sourceCtx, sourceCancel := context.WithCancel(context.Background())

		ringBuffer := ringbuffer.New[*videosource.ProcessedImage](1)

		go func() {
			defer sourceCancel()
			timeoutTick := time.NewTicker(time.Second * 4)
			rx := 0
			var unsubOnce sync.Once
			unsubFunc := func() {
				imagesSub.Unsubscribe()
			}
		SourceLoop:
			for {
				select {
				case <-socketCtx.Done():
					unsubOnce.Do(unsubFunc)
				case msg, ok := <-imagesSub.Ch:
					if !ok {
						break SourceLoop
					}
					img := msg.Data.(*videosource.ProcessedImage)
					rx++
					ringBuffer.Add(img)
				case <-timeoutTick.C:
					if rx == 0 {
						unsubOnce.Do(unsubFunc)
						break SourceLoop
					}
					rx = 0
				}
			}
			timeoutTick.Stop()
		}()

		receive := func(msgType int, data []byte) {
			// Nothing
		}
		send := func(ctx context.Context, c *websocket.Conn) {
		SendLoop:
			for {
				select {
				case <-ctx.Done():
					break SendLoop
				case <-sourceCtx.Done():
					remainingImgs := ringBuffer.GetAll()
					needCleanup := false
					for _, img := range remainingImgs {
						if needCleanup {
							img.Cleanup()
						} else if !writeOut(c, img, width, jpegQuality) {
							// bad write so cleanup the remaining
							needCleanup = true
						}
					}
					c.Close()
					break SendLoop
				case img, ok := <-ringBuffer.GetChan():
					if !writeOut(c, img, width, jpegQuality) {
						break SendLoop
					}
					if !ok {
						break SendLoop
					}
				}
			}
		}
		cleanup := func() {
			ringBuffer.Stop()
			log.Infoln("Websocket closed", websocketName)
		}

		websockets.Run(socketCtx, socketCancel, c, receive, send, cleanup)
	})
}

func writeOut(c *websocket.Conn,
	img *videosource.ProcessedImage,
	width int, jpegQuality int) (ok bool) {
	if !img.Original.IsFilled() {
		img.Cleanup()
		return true
	}
	highlighted := img.HighlightedAll()
	selectedImage := highlighted.ScaleToWidth(width)
	highlighted.Cleanup()
	imgArray := selectedImage.EncodedQuality(jpegQuality)
	selectedImage.Cleanup()
	img.Cleanup()
	zipped := gzip.Encode(imgArray, nil)
	err := c.WriteMessage(websocket.BinaryMessage, zipped)
	return err == nil
}

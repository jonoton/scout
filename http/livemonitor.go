package http

import (
	"context"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	fiber "github.com/gofiber/fiber/v2"
	websocket "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/jonoton/go-websockets"
	"github.com/jonoton/scout/gzip"
	"github.com/jonoton/scout/videosource"
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

		uuid := uuid.New().String()
		monitorName := localsMonName.(string)

		width, err := strconv.Atoi(localsWidth.(string))
		if err != nil {
			width = 0
		}
		jpegQuality, err := strconv.Atoi(localsJpegQuality.(string))
		if err != nil {
			jpegQuality = 60
		}

		log.Infoln("Websocket opened", uuid)
		socketCtx, socketCancel := context.WithCancel(context.Background())
		sourceCtx, sourceCancel := context.WithCancel(context.Background())

		ringBuffer := videosource.NewRingBufferProcessedImage(1)
		images := h.manage.Subscribe(monitorName, uuid+"-live-"+monitorName)
		go func() {
			defer sourceCancel()
			timeoutTick := time.NewTicker(time.Second * 4)
			rx := 0
			var unsubOnce sync.Once
			unsubFunc := func() {
				h.manage.Unsubscribe(monitorName, uuid+"-live-"+monitorName)
			}
		SourceLoop:
			for {
				select {
				case <-socketCtx.Done():
					unsubOnce.Do(unsubFunc)
				case img, ok := <-images:
					if !ok {
						break SourceLoop
					}
					rx++
					popped := ringBuffer.Push(img)
					popped.Cleanup()
				case <-timeoutTick.C:
					if rx == 0 {
						unsubOnce.Do(unsubFunc)
						break SourceLoop
					}
					rx = 0
				}
			}
			timeoutTick.Stop()
			if socketCtx.Err() != nil {
				cleanupRingBuffer(ringBuffer)
			}
		}()

		receive := func(msgType int, data []byte) {
			// Nothing
		}
		send := func(ctx context.Context, c *websocket.Conn) {
		SendLoop:
			for {
				select {
				case <-ctx.Done():
					cleanupRingBuffer(ringBuffer)
					break SendLoop
				case <-sourceCtx.Done():
					for ringBuffer.Len() != 0 {
						if !writeOut(c, ringBuffer, width, jpegQuality) {
							break
						}
					}
					c.Close()
					cleanupRingBuffer(ringBuffer)
					break SendLoop
				case _, ok := <-ringBuffer.Ready():
					if !ok {
						break SendLoop
					}
					if !writeOut(c, ringBuffer, width, jpegQuality) {
						break SendLoop
					}
				}
			}
		}
		cleanup := func() {
			cleanupRingBuffer(ringBuffer)
			log.Infoln("Websocket closed", uuid)
		}

		websockets.Run(socketCtx, socketCancel, c, receive, send, cleanup)
	})
}

func writeOut(c *websocket.Conn,
	ringBuffer *videosource.RingBufferProcessedImage,
	width int, jpegQuality int) (ok bool) {
	img := ringBuffer.Pop()
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

func cleanupRingBuffer(ringBuffer *videosource.RingBufferProcessedImage) {
	for ringBuffer.Len() > 0 {
		img := ringBuffer.Pop()
		img.Cleanup()
	}
}

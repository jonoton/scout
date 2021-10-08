package http

import (
	"strconv"

	log "github.com/sirupsen/logrus"

	fiber "github.com/gofiber/fiber/v2"
	websocket "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/jonoton/scout/gzip"
	"github.com/jonoton/scout/http/websockets"
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
		socketClosed := make(chan bool)
		sourceDone := make(chan bool)
		ringBuffer := videosource.NewRingBufferProcessedImage(1)
		images := h.manage.Subscribe(monitorName, uuid+"-live-"+monitorName)
		go func() {
			for img := range images {
				popped := ringBuffer.Push(img)
				popped.Cleanup()
			}
			close(sourceDone)
		}()

		receive := func(msgType int, data []byte) {
			// Nothing
		}
		send := func(c *websocket.Conn) {
			mySend(h, c, socketClosed, sourceDone, ringBuffer, uuid,
				monitorName, width, jpegQuality)
		}
		cleanup := func() {
			myCleanup(uuid, ringBuffer)
		}

		websockets.Run(c, socketClosed, receive, send, cleanup)
	})
}

func mySend(h *Http, c *websocket.Conn,
	socketClosed chan bool, sourceDone chan bool,
	ringBuffer *videosource.RingBufferProcessedImage,
	uuid string, monitorName string,
	width int, jpegQuality int) {
Loop:
	for {
		select {
		case <-socketClosed:
			h.manage.Unsubscribe(monitorName, uuid+"-live-"+monitorName)
			break Loop
		case <-sourceDone:
			if ringBuffer.Len() == 0 {
				break Loop
			}
			for ringBuffer.Len() != 0 {
				if !writeOut(h, c, ringBuffer, uuid, monitorName,
					width, jpegQuality) {
					break Loop
				}
			}
			break Loop
		case _, ok := <-ringBuffer.Ready():
			if !ok {
				break Loop
			}
			if !writeOut(h, c, ringBuffer, uuid, monitorName,
				width, jpegQuality) {
				break Loop
			}
		}
	}
}

func writeOut(h *Http, c *websocket.Conn,
	ringBuffer *videosource.RingBufferProcessedImage,
	uuid string, monitorName string,
	width int, jpegQuality int) (ok bool) {
	img := ringBuffer.Pop()
	if !img.Original.IsFilled() {
		img.Cleanup()
		return true
	}
	var selectedImage videosource.Image
	if img.HighlightedFace.IsFilled() {
		selectedImage = img.HighlightedFace.ScaleToWidth(width)
	} else if img.HighlightedObject.IsFilled() {
		selectedImage = img.HighlightedObject.ScaleToWidth(width)
	} else if img.HighlightedMotion.IsFilled() {
		selectedImage = img.HighlightedMotion.ScaleToWidth(width)
	} else {
		selectedImage = img.Original.ScaleToWidth(width)
	}
	imgArray := selectedImage.EncodedQuality(jpegQuality)
	selectedImage.Cleanup()
	img.Cleanup()
	zipped := gzip.Encode(imgArray, nil)
	err := c.WriteMessage(websocket.BinaryMessage, zipped)
	if err != nil {
		// socket closed
		h.manage.Unsubscribe(monitorName, uuid+"-live-"+monitorName)
		return false
	}
	return true
}

func myCleanup(uuid string, ringBuffer *videosource.RingBufferProcessedImage) {
	for ringBuffer.Len() > 0 {
		img := ringBuffer.Pop()
		img.Cleanup()
	}
	log.Infoln("Websocket closed", uuid)
}

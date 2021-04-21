package http

import (
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"

	fiber "github.com/gofiber/fiber/v2"
	websocket "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/jonoton/scout/gzip"
	"github.com/jonoton/scout/http/websockets"
	"github.com/jonoton/scout/sharedmat"
	"github.com/jonoton/scout/videosource"
	"gocv.io/x/gocv"
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
		images := h.manage.Subscribe(monitorName, uuid+"-live-"+monitorName)

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

		receive := func(msgType int, data []byte) {
			// TBD
			// log.Infoln("Read Func called")
		}
		send := func(c *websocket.Conn) {
			wg := sync.WaitGroup{}
			wg.Add(2)
			go func() {
				defer wg.Done()
				defer close(sourceDone)
				for {
					select {
					case img, ok := <-images:
						if !ok {
							img.Cleanup()
							return
						}
						popped := ringBuffer.Push(img)
						popped.Cleanup()
					}
				}
			}()
			go func() {
				defer wg.Done()
				writeOut := func() (ok bool) {
					img := ringBuffer.Pop()
					if !img.Original.IsValid() {
						img.Cleanup()
						return true
					}
					var imgArray []byte
					jpgParams := []int{gocv.IMWriteJpegQuality, jpegQuality}
					var selectedImage videosource.Image
					if img.HighlightedFace.IsValid() {
						selectedImage = img.HighlightedFace
					} else if img.HighlightedObject.IsValid() {
						selectedImage = img.HighlightedObject
					} else if img.HighlightedMotion.IsValid() {
						selectedImage = img.HighlightedMotion
					} else {
						selectedImage = img.Original
					}
					scaledImage := selectedImage.ScaleToWidth(width)
					if scaledImage.SharedMat != nil {
						scaledImage.SharedMat.Guard.RLock()
						if sharedmat.Valid(&scaledImage.SharedMat.Mat) {
							encoded, _ := gocv.IMEncodeWithParams(gocv.JPEGFileExt, scaledImage.SharedMat.Mat, jpgParams)
							imgArray = encoded
						}
						scaledImage.SharedMat.Guard.RUnlock()
					}
					scaledImage.Cleanup()
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
				for {
					select {
					case <-socketClosed:
						h.manage.Unsubscribe(monitorName, uuid+"-live-"+monitorName)
						return
					case <-sourceDone:
						if ringBuffer.Len() == 0 {
							return
						}
						for ringBuffer.Len() != 0 {
							if !writeOut() {
								return
							}
						}
					case _, ok := <-ringBuffer.Ready():
						if !ok {
							return
						}
						if !writeOut() {
							return
						}
					}
				}
			}()
			wg.Wait()
		}
		cleanup := func() {
			for ringBuffer.Len() > 0 {
				img := ringBuffer.Pop()
				img.Cleanup()
			}
			log.Infoln("Websocket closed", uuid)
		}

		websockets.Run(c, socketClosed, receive, send, cleanup)
	})
}

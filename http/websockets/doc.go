/*
Package websockets is a wrapper for gofiber websocket.

Example:

  h.fiber.Get("/ws", websocket.New(func(c *websocket.Conn) {
	// get locals to use
	myLocal := c.Locals("myLocal")
	if myLocal == nil {
		log.Println("No myLocal")
		return
	}
	log.Println("Websocket opened")

	socketClosed := make(chan bool)
	sourceDone := make(chan bool)

	// use or implement custom buffer so non blocking
	// see videosource.RingBufferProcessedImage for an example which allows for drops
	// some services must guarantee msgs sent, use appropriate buffer type
	ringBuffer := customringbuffer.NewRingBuffer(1)
	stuffChan := h.myEngine.Subscribe("stuff")
	go func() {
		for cur := range stuffChan {
			popped := ringBuffer.Push(cur)
			if popped.IsValid() {
				log.Println("Dropped stuff at websocket, but that's expected/ok")
			}
			popped.Cleanup()
		}
		close(sourceDone)
	}()

	receive := func(msgType int, data []byte) {
		log.Println("Read Func called")
	}
	send := func(c *websocket.Conn) {
		writeOut := func() (ok bool) {
			stuff := ringBuffer.Pop()
			if !stuff.IsValid() {
				stuff.Cleanup()
				return true
			}
			var stuffArray []byte
			// serialize stuff into stuffAray
			stuff.Cleanup()
			err := c.WriteMessage(websocket.BinaryMessage, stuffArray)
			if err != nil {
				// socket closed
				h.myEngine.Unsubscribe("stuff")
				return false
			}
			return true
		}
	Loop:
		for {
			select {
			case <-socketClosed:
				h.myEngine.Unsubscribe("stuff")
				break Loop
			case <-sourceDone:
				if ringBuffer.Len() == 0 {
					break Loop
				}
				for ringBuffer.Len() != 0 {
					if !writeOut() {
						break Loop
					}
				}
			case _, ok := <-ringBuffer.Ready():
				if !ok {
					break Loop
				}
				if !writeOut() {
					break Loop
				}
			}
		}
	}
	cleanup := func() {
		for ringBuffer.Len() > 0 {
			stuff := ringBuffer.Pop()
			stuff.Cleanup()
		}
		log.Println("Websocket closed")
	}

	websockets.Run(c, socketClosed, receive, send, cleanup)
  }))
*/
package websockets

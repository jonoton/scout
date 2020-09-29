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

	stuffChan := h.myEngine.Subscribe("stuff")

	socketClosed := make(chan bool)
	sourceDone := make(chan bool)
	// use or implement custom buffer so non blocking
	// see videosource.RingBufferProcessedImage for an example which allows for drops
	// some services must guarantee msgs sent, use appropriate buffer type
	ringBuffer := customringbuffer.NewRingBuffer(10)

	receive := func(msgType int, data []byte) {
		log.Println("Read Func called")
		// do something here with msgs
	}
	send := func(c *websocket.Conn) {
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			defer close(sourceDone)
			for {
				select {
				case cur, ok := <-stuffChan:
					if !ok {
						return
					}
					popped := ringBuffer.Push(cur)
					if popped.IsValid() {
						log.Println("Dropped stuff at websocket")
					}
					popped.Cleanup()
				}
			}
		}()
		go func() {
			defer wg.Done()
			writeOut := func() (ok bool) {
					stuff := ringBuffer.Pop()
					if !stuff.IsValid() {
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
			for {
					select {
					case <-socketClosed:
						h.myEngine.Unsubscribe("stuff")
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
			stuff := ringBuffer.Pop()
			stuff.Cleanup()
		}
		log.Println("Websocket closed")
	}

	websockets.Run(c, socketClosed, receive, send, cleanup)
  }))
*/
package websockets

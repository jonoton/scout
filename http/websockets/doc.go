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

	socketCtx, socketCancel := context.WithCancel(context.Background())
	sourceCtx, sourceCancel := context.WithCancel(context.Background())

	// use or implement custom buffer so non blocking
	// see videosource.RingBufferProcessedImage for an example which allows for drops
	// some services must guarantee msgs sent, use appropriate buffer type
	ringBuffer := customringbuffer.NewRingBuffer(1)
	stuffChan := h.myEngine.Subscribe("stuff")
	go func() {
		defer sourceCancel()
		// for select 
		//   read stuffChan and push into ringBuffer
		//   unsubscribe from stuff if socketCtx.Done() or stale
	}()

	receive := func(msgType int, data []byte) {
		log.Println("Read Func called")
	}
	send := func(c *websocket.Conn) {
		defer socketCancel()
		// for select 
		//   use ring buffer to write out to socket
		//   check sourceCtx.Done() and send all ring buffer then cleanup
	}
	cleanup := func() {
		// cleanup ring buffer
		log.Println("Websocket closed")
	}

	websockets.Run(socketCtx, c, receive, send, cleanup)
  }))
*/
package websockets

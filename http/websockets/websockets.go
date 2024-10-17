package websockets

import (
	"context"
	"sync"

	websocket "github.com/gofiber/websocket/v2"
)

// Run is a wrapper for gofiber websocket
//
//	socketCtx - provides a way to gracefully shutdown goroutines when the websocket connection closes
//	receive - do not block as it will continuously run until the websocket is closed
//	send - push out msgs and return when server is done with the websocket
//	cleanup - will be called when all goroutines are finished
func Run(socketCtx context.Context, c *websocket.Conn, receive func(int, []byte), send func(*websocket.Conn), cleanup func()) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// receive goroutine will cleanup after websocket closes no need to wait for it
	go func() {
	ReceiveLoop:
		for {
			// Check if the context is done (e.g., due to websocket closing).
			if socketCtx.Err() != nil {
				break ReceiveLoop
			}
			msgType, data, err := c.ReadMessage()
			if err != nil {
				// socket closed
				break ReceiveLoop
			}
			if receive != nil && socketCtx.Err() == nil {
				receive(msgType, data)
			}
		}
	}()

	// send goroutine
	go func() {
		defer wg.Done()
		if send != nil {
			send(c)
		}
	}()

	wg.Wait()
	if cleanup != nil {
		cleanup()
	}
}

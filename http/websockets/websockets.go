package websockets

import (
	"context"
	"sync"

	websocket "github.com/gofiber/websocket/v2"
)

// Run is a wrapper for gofiber websocket
//
//	socketCtx - provides a way to gracefully shutdown goroutines when the websocket connection closes
//	socketCancel - cancels the socketCtx when errors occur or done
//	receive - do not block as it will continuously run until the websocket handler reaches the end
//	send - push out msgs and return when server is done with the websocket
//	cleanup - will be called when all goroutines are finished
func Run(socketCtx context.Context, socketCancel context.CancelFunc, c *websocket.Conn, receive func(int, []byte), send func(context.Context, *websocket.Conn), cleanup func()) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// read goroutine
	go func() {
		defer socketCancel()
	ReceiveLoop:
		for {
			// Check if the context is done (e.g., due to websocket closing).
			if socketCtx.Err() != nil || c.Conn == nil {
				break ReceiveLoop
			}
			// ReadMessage will return when the websocket is cleaned up
			// Note: c.Close() does not currently clean up hijacked connections
			//   such as this websocket handler. Do not wait on ReadMessage as that
			//   will cause this websockets.Run handler to block forever.
			//   The hijack connection will be cleaned up after the websocket.New
			//   handler reaches the end.
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
		defer socketCancel()
		if send != nil {
			send(socketCtx, c)
		}
	}()

	wg.Wait()
	if cleanup != nil {
		cleanup()
	}
}

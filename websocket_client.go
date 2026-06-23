package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func streamWebSocket(ctx context.Context, bcsvrkey string, outbound chan<- []byte) error {
	u := url.URL{Scheme: "wss", Host: "online.showroom-live.com", Path: "/"}
	log.Printf("connecting websocket: %s", u.String())

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shutdown"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
	}()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("SUB\t"+bcsvrkey)); err != nil {
		return fmt.Errorf("send subscription: %w", err)
	}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read websocket message: %w", err)
		}

		if messageType != websocket.TextMessage {
			continue
		}

		copied := append([]byte(nil), message...)
		select {
		case outbound <- copied:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func streamWebSocket(ctx context.Context, bcsvrkey string, outbound chan<- []byte) error {
	if bcsvrkey == "" {
		return fmt.Errorf("empty bcsvrkey")
	}

	u := url.URL{Scheme: "wss", Host: "online.showroom-live.com", Path: "/"}
	backoff := 2 * time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		log.Printf("connecting websocket: %s", u.String())

		headers := http.Header{}
		headers.Set("Origin", "https://www.showroom-live.com")
		headers.Set("Cache-Control", "no-cache")
		headers.Set("Pragma", "no-cache")
		headers.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36")
		headers.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")

		dialer := *websocket.DefaultDialer
		dialer.EnableCompression = true
		conn, _, err := dialer.DialContext(ctx, u.String(), headers)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("dial websocket failed: %v", err)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
			if backoff < 30*time.Second {
				backoff *= 2
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
			}
			continue
		}

		err = func() error {
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
		}()

		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("websocket session ended: %v", err)
		}

		backoff = 2 * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

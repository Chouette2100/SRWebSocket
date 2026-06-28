package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func streamWebSocket(ctx context.Context, bcsvrkey string, outbound chan<- []byte) error {
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

		func() {
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
			for {
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Printf("read websocket message failed: %v", err)
					return
				}

				if messageType != websocket.TextMessage {
					continue
				}

				copied := append([]byte(nil), message...)
				select {
				case outbound <- copied:
				case <-ctx.Done():
					return
				}
			}
		}()

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

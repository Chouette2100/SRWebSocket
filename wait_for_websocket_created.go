package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var roomPageFetcher = http.DefaultClient

var websocketDialer = websocket.DefaultDialer

var websocketEndpoint = "wss://online.showroom-live.com/"

var roomTokenPattern = regexp.MustCompile(`"[0-9a-f]{64}"`)

type websocketCreatedEnvelope struct {
	Type      int   `json:"t"`
	CreatedAt int64 `json:"created_at"`
}

func WaitForWebSocketCreated(roomurl string) (time.Time, error) {
	token, err := fetchRoomToken(roomurl)
	if err != nil {
		return time.Time{}, err
	}

	headers := http.Header{}
	headers.Set("Origin", "https://www.showroom-live.com")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Pragma", "no-cache")
	headers.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36")
	headers.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")

	dialer := *websocketDialer
	dialer.EnableCompression = true
	conn, _, err := dialer.Dial(websocketEndpoint, headers)
	if err != nil {
		return time.Time{}, fmt.Errorf("dial websocket: %w", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("SUB\t"+token)); err != nil {
		return time.Time{}, fmt.Errorf("send subscription: %w", err)
	}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			return time.Time{}, fmt.Errorf("read websocket message: %w", err)
		}
		if messageType != websocket.TextMessage {
			continue
		}

		if ct, ok := parseWebSocketCreatedMessage(message); ok {
			return ct, nil
		}
	}
}

func fetchRoomToken(roomurl string) (string, error) {
	response, err := roomPageFetcher.Get(roomurl)
	if err != nil {
		return "", fmt.Errorf("fetch room page: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read room page: %w", err)
	}

	match := roomTokenPattern.Find(body)
	if match == nil {
		return "", fmt.Errorf("room token not found in page: %s", roomurl)
	}

	return string(match[1 : len(match)-1]), nil
}

func parseWebSocketCreatedMessage(message []byte) (time.Time, bool) {
	parts := strings.SplitN(string(message), "\t", 3)
	if len(parts) != 3 {
		return time.Time{}, false
	}
	if !strings.HasPrefix(parts[0], "MSG") {
		return time.Time{}, false
	}

	var envelope websocketCreatedEnvelope
	if err := json.Unmarshal([]byte(parts[2]), &envelope); err != nil {
		return time.Time{}, false
	}
	if envelope.Type != 104 || envelope.CreatedAt <= 0 {
		return time.Time{}, false
	}

	return time.Unix(envelope.CreatedAt, 0), true
}

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestFetchRoomToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"</body></html>`)
	}))
	t.Cleanup(server.Close)

	oldClient := roomPageFetcher
	roomPageFetcher = server.Client()
	t.Cleanup(func() { roomPageFetcher = oldClient })

	token, err := fetchRoomToken(server.URL)
	if err != nil {
		t.Fatalf("fetchRoomToken returned error: %v", err)
	}

	want := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if token != want {
		t.Fatalf("token mismatch: got %q want %q", token, want)
	}
}

func TestWaitForWebSocketCreated(t *testing.T) {
	token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	roomServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `<html><body>"%s"</body></html>`, token)
	}))
	t.Cleanup(roomServer.Close)

	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read SUB: %v", err)
		}
		if messageType != websocket.TextMessage {
			t.Fatalf("unexpected message type: %d", messageType)
		}
		if string(message) != "SUB\t"+token {
			t.Fatalf("unexpected SUB payload: %q", string(message))
		}

		createdAt := time.Now().Unix()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("MSG\t%s\t{\"t\":104,\"created_at\":%d}", token, createdAt))); err != nil {
			t.Fatalf("write message: %v", err)
		}
	}))
	t.Cleanup(wsServer.Close)

	oldClient := roomPageFetcher
	oldDialer := websocketDialer
	oldEndpoint := websocketEndpoint
	roomPageFetcher = roomServer.Client()
	websocketDialer = &websocket.Dialer{}
	websocketEndpoint = "ws" + wsServer.URL[len("http"):]
	t.Cleanup(func() {
		roomPageFetcher = oldClient
		websocketDialer = oldDialer
		websocketEndpoint = oldEndpoint
	})

	if ct, err := WaitForWebSocketCreated(roomServer.URL); err != nil {
		t.Fatalf("WaitForWebSocketCreated returned error: %v", err)
	} else {
		if ct.IsZero() {
			t.Fatal("WaitForWebSocketCreated returned zero time")
		}
		t.Logf("WebSocket created at: %v", ct)
	}
}

func TestWebSocketCreatedMessagePattern(t *testing.T) {
	message := []byte("MSG\t0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\t{\"t\":104,\"created_at\":1782268214}")
	createdAt, ok := parseWebSocketCreatedMessage(message)
	if !ok {
		t.Fatalf("message should parse %q", string(message))
	}
	if createdAt.Unix() != 1782268214 {
		t.Fatalf("created_at mismatch: got %d", createdAt.Unix())
	}
}

func TestFetchRoomTokenRejectsMissingToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `<html><body>missing token</body></html>`)
	}))
	t.Cleanup(server.Close)

	oldClient := roomPageFetcher
	roomPageFetcher = server.Client()
	t.Cleanup(func() { roomPageFetcher = oldClient })

	_, err := fetchRoomToken(server.URL)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !regexp.MustCompile(`room token not found`).MatchString(err.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
}

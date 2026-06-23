package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var indexTemplate = template.Must(template.ParseFiles(filepath.Join("templates", "handleIndex.gtpl")))
var giftRowTemplate = template.Must(template.ParseFiles(filepath.Join("templates", "handleGiftEvents.gtpl")))

type indexTemplateData struct {
	MaxRows int
	RoomID  int
}

type giftRowData struct {
	GiftMessage
	CreatedAtDisplay string
	GiftName         string
	GiftPoint        int
	GiftFree         bool
}

func newHTTPHandler(manager *RoomManager) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/events/gifts", handleGiftEvents(manager))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	roomID, err := parseRoomID(r)
	if err != nil {
		http.Error(w, "roomid query is required, e.g. /?roomid=12345", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := indexTemplate.ExecuteTemplate(w, "handleIndex.gtpl", indexTemplateData{MaxRows: maxTableRows, RoomID: roomID}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleGiftEvents(manager *RoomManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID, err := parseRoomID(r)
		if err != nil {
			http.Error(w, "roomid query is required, e.g. /events/gifts?roomid=12345", http.StatusBadRequest)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		subscription, giftCatalog, err := manager.Subscribe(roomID)
		if err != nil {
			if errors.Is(err, r.Context().Err()) {
				return
			}
			http.Error(w, fmt.Sprintf("failed to subscribe roomid=%d: %v", roomID, err), http.StatusBadGateway)
			return
		}
		defer func() {
			manager.Unsubscribe(roomID, subscription.id)
		}()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		for _, gift := range subscription.history {
			if err := writeSSEEvent(w, gift, giftCatalog); err != nil {
				return
			}
		}
		flusher.Flush()

		keepAlive := time.NewTicker(20 * time.Second)
		defer keepAlive.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-keepAlive.C:
				if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
					return
				}
				flusher.Flush()
			case gift, ok := <-subscription.stream:
				if !ok {
					return
				}
				if err := writeSSEEvent(w, gift, giftCatalog); err != nil {
					log.Printf("sse write error: %v", err)
					return
				}
				flusher.Flush()
			}
		}
	}
}

func parseRoomID(r *http.Request) (int, error) {
	roomIDString := strings.TrimSpace(r.URL.Query().Get("roomid"))
	if roomIDString == "" {
		return 0, errors.New("roomid is required")
	}

	roomID, err := strconv.Atoi(roomIDString)
	if err != nil || roomID <= 0 {
		return 0, fmt.Errorf("invalid roomid: %q", roomIDString)
	}
	return roomID, nil
}

func writeSSEEvent(w http.ResponseWriter, gift GiftMessage, giftCatalog map[int]GiftCatalogItem) error {
	row, err := renderGiftRow(gift, giftCatalog)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprint(w, "event: gift_update\n"); err != nil {
		return err
	}
	for _, line := range strings.Split(row, "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, "\n")
	return err
}

func renderGiftRow(gift GiftMessage, giftCatalog map[int]GiftCatalogItem) (string, error) {
	var buffer bytes.Buffer

	meta := giftCatalog[gift.GiftCode]
	rowData := giftRowData{
		GiftMessage:      gift,
		CreatedAtDisplay: gift.CreatedAtTime().Format("2006-01-02 15:04:05"),
		GiftName:         meta.GiftName,
		GiftPoint:        meta.Point,
		GiftFree:         meta.Free,
	}

	if err := giftRowTemplate.ExecuteTemplate(&buffer, "handleGiftEvents.gtpl", rowData); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

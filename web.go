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
	"sync"
	"time"
)

var sumMap sync.Map // roomID -> int (累計ポイント)

var indexTemplate = template.Must(template.ParseFiles(filepath.Join("templates", "handleIndex.gtpl")))
var adminTemplate = template.Must(template.ParseFiles(filepath.Join("templates", "handleAdmin.gtpl")))
var giftRowTemplate = template.Must(template.ParseFiles(filepath.Join("templates", "handleGiftEvents.gtpl")))

type indexTemplateData struct {
	MaxRows int
	RoomID  int
}

type adminRoomRow struct {
	RoomConfig
	Status      string
	Subscribers int
	Running     bool
}

type adminTemplateData struct {
	Rooms []adminRoomRow
}

type giftRowData struct {
	GiftMessage
	CreatedAtDisplay string
	GiftName         string
	GiftPoint        int
	GiftFree         bool
	Pt               int
	Sum              int
}

func newHTTPHandler(manager *RoomManager) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/admin", handleAdmin(manager))
	mux.HandleFunc("/admin/rooms", handleAdminRooms(manager))
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

func handleAdmin(manager *RoomManager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		configs, err := LoadRoomConfigs()
		if err != nil {
			http.Error(w, fmt.Sprintf("load room configs failed: %v", err), http.StatusInternalServerError)
			return
		}

		runtime := map[int]RoomRuntimeSnapshot{}
		for _, item := range manager.Snapshot() {
			runtime[item.RoomID] = item
		}

		rows := make([]adminRoomRow, 0, len(configs))
		for _, config := range configs {
			item := adminRoomRow{RoomConfig: config}
			if current, ok := runtime[config.RoomID]; ok {
				item.Subscribers = current.Subscribers
				item.Status = current.Phase
				item.Mode = current.Mode
				item.Running = current.Phase != "停止中"
			} else {
				item.Status = "停止中"
			}
			rows = append(rows, item)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := adminTemplate.ExecuteTemplate(w, "handleAdmin.gtpl", adminTemplateData{Rooms: rows}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func handleAdminRooms(manager *RoomManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("parse form failed: %v", err), http.StatusBadRequest)
			return
		}

		roomID, err := strconv.Atoi(strings.TrimSpace(r.FormValue("roomid")))
		if err != nil || roomID <= 0 {
			http.Error(w, "invalid roomid", http.StatusBadRequest)
			return
		}

		action := r.FormValue("action")
		saveData := r.FormValue("save_data") == "on"
		mode := RoomRunMode(r.FormValue("mode"))
		if !mode.Valid() {
			mode = RoomRunModeOnce
		}

		var opErr error
		switch action {
		case "start":
			opErr = manager.StartRoomWithConfig(roomID, mode, saveData)
		case "stop":
			manager.StopRoom(roomID)
		case "start-always":
			opErr = manager.StartRoomWithConfig(roomID, RoomRunModeAlways, saveData)
		case "stop-once":
			opErr = manager.UpdateRoomMode(roomID, RoomRunModeOnce)
			manager.StopRoom(roomID)
		case "stop-always":
			opErr = manager.UpdateRoomMode(roomID, RoomRunModeOnce)
			manager.StopRoom(roomID)
		case "mode-once":
			opErr = manager.UpdateRoomMode(roomID, RoomRunModeOnce)
		case "delete":
			opErr = manager.DeleteRoom(roomID)
		default:
			opErr = manager.StartRoomWithConfig(roomID, mode, saveData)
		}
		if opErr != nil {
			http.Error(w, fmt.Sprintf("room operation failed: %v", opErr), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
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

		subscription, _, err := manager.Subscribe(roomID)
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
			if err := writeSSEEvent(w, gift); err != nil {
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
				if err := writeSSEEvent(w, gift); err != nil {
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

func writeSSEEvent(w http.ResponseWriter, row giftRowData) error {
	rowText, err := renderGiftRow(row)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprint(w, "event: gift_update\n"); err != nil {
		return err
	}
	for _, line := range strings.Split(rowText, "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, "\n")
	return err
}

func buildGiftRowData(roomid int, gift GiftMessage, giftCatalog map[int]GiftCatalogItem) giftRowData {
	meta := giftCatalog[gift.GiftCode]

	pt := meta.Point * gift.Count
	if gift.Count >= 10 {
		pt = int(float64(pt) * 1.2)
	}
	if gift.H == 1 {
		pt = int(float64(pt) * 2.5)
	}

	currentSum := 0
	if cpt, ok := sumMap.Load(roomid); ok {
		currentSum, _ = cpt.(int)
	}
	newSum := currentSum + pt
	sumMap.Store(roomid, newSum)

	return giftRowData{
		GiftMessage:      gift,
		CreatedAtDisplay: gift.CreatedAtTime().Format("2006-01-02 15:04:05"),
		GiftName:         meta.GiftName,
		GiftPoint:        meta.Point,
		GiftFree:         meta.Free,
		Pt:               pt,
		Sum:              newSum,
	}
}

func renderGiftRow(row giftRowData) (string, error) {
	var buffer bytes.Buffer
	if err := giftRowTemplate.ExecuteTemplate(&buffer, "handleGiftEvents.gtpl", row); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

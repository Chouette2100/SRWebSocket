package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type subscribeResponse struct {
	subscription subscription
	catalog      map[int]GiftCatalogItem
	err          error
}

type roomWorker struct {
	roomID              int
	roomURL             string
	catalog             map[int]GiftCatalogItem
	rawMessages         chan []byte
	subscribeRequests   chan subscriptionRequest
	unsubscribeRequests chan int
	cancel              context.CancelFunc
}

type roomState struct {
	worker      *roomWorker
	subscribers int
	phase       string
	mode        RoomRunMode
	saveData    bool
	idleTimer   *time.Timer
}

type RoomRuntimeSnapshot struct {
	RoomID      int
	Subscribers int
	Phase       string
	Mode        RoomRunMode
}

type RoomManager struct {
	ctx         context.Context
	idleTimeout time.Duration

	mu    sync.Mutex
	rooms map[int]*roomState
}

func NewRoomManager(ctx context.Context, idleTimeout time.Duration) *RoomManager {
	if idleTimeout <= 0 {
		idleTimeout = 2 * time.Minute
	}
	return &RoomManager{
		ctx:         ctx,
		idleTimeout: idleTimeout,
		rooms:       make(map[int]*roomState),
	}
}

func (m *RoomManager) Subscribe(roomID int) (subscription, map[int]GiftCatalogItem, error) {
	if roomID <= 0 {
		return subscription{}, nil, fmt.Errorf("invalid roomid: %d", roomID)
	}

	m.mu.Lock()
	state, ok := m.rooms[roomID]
	if !ok {
		worker, err := m.startWorkerLocked(roomID)
		if err != nil {
			m.mu.Unlock()
			return subscription{}, nil, err
		}
		state = &roomState{worker: worker, phase: "データ取得待ち", mode: RoomRunModeOnce}
		m.rooms[roomID] = state
	} else if state.worker == nil {
		worker, err := m.startWorkerLocked(roomID)
		if err != nil {
			m.mu.Unlock()
			return subscription{}, nil, err
		}
		state.worker = worker
		state.phase = "データ取得待ち"
		if !state.mode.Valid() {
			state.mode = RoomRunModeOnce
		}
	}

	if state.idleTimer != nil {
		state.idleTimer.Stop()
		state.idleTimer = nil
	}
	m.mu.Unlock()

	response := make(chan subscription)
	request := subscriptionRequest{response: response}

	select {
	case state.worker.subscribeRequests <- request:
	case <-m.ctx.Done():
		return subscription{}, nil, m.ctx.Err()
	}

	var sub subscription
	select {
	case sub = <-response:
	case <-m.ctx.Done():
		return subscription{}, nil, m.ctx.Err()
	}

	m.mu.Lock()
	state.subscribers++
	catalog := state.worker.catalog
	m.mu.Unlock()

	return sub, catalog, nil
}

func (m *RoomManager) Unsubscribe(roomID int, subscriberID int) {
	m.mu.Lock()
	state, ok := m.rooms[roomID]
	if !ok {
		m.mu.Unlock()
		return
	}
	worker := state.worker
	m.mu.Unlock()

	select {
	case worker.unsubscribeRequests <- subscriberID:
	case <-time.After(250 * time.Millisecond):
	case <-m.ctx.Done():
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok = m.rooms[roomID]
	if !ok {
		return
	}
	if state.subscribers > 0 {
		state.subscribers--
	}
	if state.mode == RoomRunModeAlways || state.subscribers != 0 || state.idleTimer != nil {
		return
	}

	state.idleTimer = time.AfterFunc(m.idleTimeout, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		current, stillExists := m.rooms[roomID]
		if !stillExists || current.mode == RoomRunModeAlways || current.subscribers > 0 {
			if stillExists {
				current.idleTimer = nil
			}
			return
		}

		log.Printf("stopping idle room worker: roomid=%d", roomID)
		current.worker.cancel()
		current.worker = nil
		current.phase = "停止中"
		delete(m.rooms, roomID)
	})
}

func (m *RoomManager) SetRoomPhase(roomID int, phase string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.rooms[roomID]
	if !ok {
		return
	}
	state.phase = phase
}

func (m *RoomManager) Snapshot() []RoomRuntimeSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]RoomRuntimeSnapshot, 0, len(m.rooms))
	for roomID, state := range m.rooms {
		result = append(result, RoomRuntimeSnapshot{
			RoomID:      roomID,
			Subscribers: state.subscribers,
			Phase:       state.phase,
			Mode:        state.mode,
		})
	}
	return result
}

func (m *RoomManager) ShouldSave(roomID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.rooms[roomID]
	if !ok {
		return false
	}
	return state.saveData
}

func (m *RoomManager) BootstrapFromConfigs(configs []RoomConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, config := range configs {
		config = config.Normalize()
		state, ok := m.rooms[config.RoomID]
		if !ok {
			m.rooms[config.RoomID] = &roomState{
				phase:    "停止中",
				mode:     config.Mode,
				saveData: config.SaveData,
			}
			continue
		}
		state.mode = config.Mode
		state.saveData = config.SaveData
		if state.worker == nil {
			state.phase = "停止中"
		}
	}
}

func (m *RoomManager) startWorkerLocked(roomID int) (*roomWorker, error) {
	catalog, err := GetGiftCatalog(roomID)
	if err != nil {
		return nil, fmt.Errorf("get gift catalog roomid=%d: %w", roomID, err)
	}

	roomURL, err := roomURLFromRoomID(roomID)
	if err != nil {
		return nil, fmt.Errorf("get room url roomid=%d: %w", roomID, err)
	}

	mode := RoomRunModeOnce
	saveData := false
	if config, err := LoadRoomConfig(roomID); err == nil {
		mode = config.Mode
		saveData = config.SaveData
	}

	workerCtx, cancel := context.WithCancel(m.ctx)
	worker := &roomWorker{
		roomID:              roomID,
		roomURL:             roomURL,
		catalog:             catalog,
		rawMessages:         make(chan []byte, 4096),
		subscribeRequests:   make(chan subscriptionRequest),
		unsubscribeRequests: make(chan int, 64),
		cancel:              cancel,
	}
	state := &roomState{worker: worker, phase: "データ取得待ち", mode: mode, saveData: saveData}
	if _, ok := m.rooms[roomID]; !ok {
		m.rooms[roomID] = state
	}

	if _, loaded := sumMap.LoadOrStore(roomID, 0); !loaded {
		log.Printf("initialized sumMap for roomid=%d", roomID)
	} else {
		sumMap.Store(roomID, 0)
	}

	go runGiftHub(workerCtx, roomID, worker.catalog, m.persistGiftEvent, worker.rawMessages, worker.subscribeRequests, worker.unsubscribeRequests)
	go runWorkerCollector(workerCtx, m, worker.roomID, worker.roomURL, worker.rawMessages)

	log.Printf("started room worker: roomid=%d gifts=%d", roomID, len(catalog))
	return worker, nil
}

func (m *RoomManager) persistGiftEvent(roomID int, row giftRowData) {
	if !m.ShouldSave(roomID) {
		return
	}
	if err := SaveGiftEvent(roomID, row); err != nil {
		log.Printf("save gift event failed: roomid=%d err=%v", roomID, err)
	}
}

func (m *RoomManager) StartRoom(roomID int, mode RoomRunMode, saveData bool) error {
	if roomID <= 0 {
		return fmt.Errorf("invalid roomid: %d", roomID)
	}
	if !mode.Valid() {
		mode = RoomRunModeOnce
	}

	config := RoomConfig{RoomID: roomID, Mode: mode, SaveData: saveData}.Normalize()
	if err := SaveRoomConfig(config); err != nil {
		return err
	}

	m.mu.Lock()
	state, ok := m.rooms[roomID]
	if ok && state.worker != nil {
		state.mode = mode
		state.saveData = saveData
		m.mu.Unlock()
		return nil
	}
	if ok && state.worker == nil {
		state.mode = mode
		state.saveData = saveData
		m.mu.Unlock()
		_, err := m.startWorkerLocked(roomID)
		if err != nil {
			return err
		}
		m.mu.Lock()
		if state, ok := m.rooms[roomID]; ok {
			state.mode = mode
			state.saveData = saveData
		}
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	_, err := m.startWorkerLocked(roomID)
	if err != nil {
		return err
	}

	m.mu.Lock()
	if state, ok := m.rooms[roomID]; ok {
		state.mode = mode
		state.saveData = saveData
	}
	m.mu.Unlock()
	return nil
}

func (m *RoomManager) StopRoom(roomID int) {
	m.mu.Lock()
	state, ok := m.rooms[roomID]
	if !ok {
		m.mu.Unlock()
		return
	}
	state.phase = "停止中"
	if state.worker != nil {
		state.worker.cancel()
		state.worker = nil
	}
	m.mu.Unlock()
}

func (m *RoomManager) DeleteRoom(roomID int) error {
	m.StopRoom(roomID)
	m.mu.Lock()
	delete(m.rooms, roomID)
	m.mu.Unlock()
	return DeleteRoomConfig(roomID)
}

func (m *RoomManager) UpdateRoomMode(roomID int, mode RoomRunMode) error {
	config, err := LoadRoomConfig(roomID)
	if err != nil {
		return err
	}
	if !mode.Valid() {
		mode = RoomRunModeOnce
	}
	config.Mode = mode
	return SaveRoomConfig(config)
}

func (m *RoomManager) StartRoomWithConfig(roomID int, mode RoomRunMode, saveData bool) error {
	return m.StartRoom(roomID, mode, saveData)
}

func runWorkerCollector(ctx context.Context, manager *RoomManager, roomID int, roomURL string, outbound chan<- []byte) {
	acquiredOnce := false
	for {
		manager.SetRoomPhase(roomID, "データ取得待ち")
		bcsvrkey, err := GetBcsvrkey(roomID)
		if err == nil {
			manager.SetRoomPhase(roomID, "データ取得中")
			// bcsvrkeyが取得できた = 配信がハイ待った = ギフトの受信を開始する
			if err := streamWebSocket(ctx, bcsvrkey, outbound); err != nil && ctx.Err() == nil {
				// エラーが発生した場合は、再度bcsvrkeyを取得して再接続する
				// 配信が終わった場合もここにくるが、その場合は再度bcsvrkeyを取得できないので、
				// 次のループで待機することになる
				log.Printf("streamWebSocket ended: roomid=%d err=%v", roomID, err)
			}
			acquiredOnce = true
			continue
		}

		log.Printf("GetBcsvrkey failed: roomid=%d err=%v", roomID, err)
		manager.SetRoomPhase(roomID, "データ取得待ち")
		createdAt, waitErr := WaitForWebSocketCreated(roomURL)
		if waitErr != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("WaitForWebSocketCreated failed: roomid=%d err=%v", roomID, waitErr)
			if state, ok := manager.rooms[roomID]; ok && state.mode == RoomRunModeOnce && acquiredOnce {
				return
			}
			continue
		}

		if ct, status := parseWebSocketCreated(createdAt); status {
			sumMap.Store(roomID, 0)
			log.Printf("websocket created detected: roomid=%d created_at=%s", roomID, ct.Format(time.RFC3339))
		}
		if state, ok := manager.rooms[roomID]; ok && state.mode == RoomRunModeOnce && acquiredOnce {
			return
		}
	}
}

func roomURLFromRoomID(roomID int) (string, error) {
	usr, err := getUserInfo(roomID)
	if err != nil {
		return "", err
	}
	return "https://www.showroom-live.com/r/" + usr.Userid, nil
}

func parseWebSocketCreated(createdAt time.Time) (time.Time, bool) {
	if createdAt.IsZero() {
		return time.Time{}, false
	}

	delta := time.Since(createdAt)
	if delta < 0 {
		delta = -delta
	}
	if delta > 2*time.Minute {
		return time.Time{}, false
	}
	return createdAt, true
}

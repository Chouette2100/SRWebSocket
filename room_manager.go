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
	bcsvrkey            string
	catalog             map[int]GiftCatalogItem
	rawMessages         chan []byte
	subscribeRequests   chan subscriptionRequest
	unsubscribeRequests chan int
	cancel              context.CancelFunc
}

type roomState struct {
	worker      *roomWorker
	subscribers int
	idleTimer   *time.Timer
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
		state = &roomState{worker: worker}
		m.rooms[roomID] = state
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
	if state.subscribers != 0 || state.idleTimer != nil {
		return
	}

	state.idleTimer = time.AfterFunc(m.idleTimeout, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		current, stillExists := m.rooms[roomID]
		if !stillExists || current.subscribers > 0 {
			if stillExists {
				current.idleTimer = nil
			}
			return
		}

		log.Printf("stopping idle room worker: roomid=%d", roomID)
		current.worker.cancel()
		delete(m.rooms, roomID)
	})
}

func (m *RoomManager) startWorkerLocked(roomID int) (*roomWorker, error) {
	bcsvrkey, err := GetBcsvrkey(roomID)
	if err != nil {
		return nil, fmt.Errorf("get bcsvrkey roomid=%d: %w", roomID, err)
	}

	catalog, err := GetGiftCatalog(roomID)
	if err != nil {
		return nil, fmt.Errorf("get gift catalog roomid=%d: %w", roomID, err)
	}

	roomURL, err := roomURLFromRoomID(roomID)
	if err != nil {
		return nil, fmt.Errorf("get room url roomid=%d: %w", roomID, err)
	}

	workerCtx, cancel := context.WithCancel(m.ctx)
	worker := &roomWorker{
		roomID:              roomID,
		roomURL:             roomURL,
		bcsvrkey:            bcsvrkey,
		catalog:             catalog,
		rawMessages:         make(chan []byte, 4096),
		subscribeRequests:   make(chan subscriptionRequest),
		unsubscribeRequests: make(chan int, 64),
		cancel:              cancel,
	}

	if _, loaded := sumMap.LoadOrStore(roomID, 0); !loaded {
		log.Printf("initialized sumMap for roomid=%d", roomID)
	} else {
		sumMap.Store(roomID, 0)
	}

	go runGiftHub(workerCtx, worker.bcsvrkey, worker.rawMessages, worker.subscribeRequests, worker.unsubscribeRequests)
	go runWorkerCollector(workerCtx, worker.roomID, worker.roomURL, worker.rawMessages)

	log.Printf("started room worker: roomid=%d bcsvrkey=%s gifts=%d", roomID, bcsvrkey, len(catalog))
	return worker, nil
}

func runWorkerCollector(ctx context.Context, roomID int, roomURL string, outbound chan<- []byte) {
	for {
		bcsvrkey, err := GetBcsvrkey(roomID)
		if err == nil {
			// bcsvrkeyが取得できた = 配信がハイ待った = ギフトの受信を開始する
			if err := streamWebSocket(ctx, bcsvrkey, outbound); err != nil && ctx.Err() == nil {
				// エラーが発生した場合は、再度bcsvrkeyを取得して再接続する
				// 配信が終わった場合もここにくるが、その場合は再度bcsvrkeyを取得できないので、
				// 次のループで待機することになる
				log.Printf("streamWebSocket ended: roomid=%d err=%v", roomID, err)
			}
			continue
		}

		log.Printf("GetBcsvrkey failed: roomid=%d err=%v", roomID, err)
		createdAt, waitErr := WaitForWebSocketCreated(roomURL)
		if waitErr != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("WaitForWebSocketCreated failed: roomid=%d err=%v", roomID, waitErr)
			continue
		}

		if ct, status := parseWebSocketCreated(createdAt); status {
			sumMap.Store(roomID, 0)
			log.Printf("websocket created detected: roomid=%d created_at=%s", roomID, ct.Format(time.RFC3339))
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

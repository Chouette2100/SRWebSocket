package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
)

func runGiftHub(
	ctx context.Context,
	roomID int,
	giftCatalog map[int]GiftCatalogItem,
	persistGift func(int, giftRowData),
	inbound <-chan []byte,
	subscribeRequests <-chan subscriptionRequest,
	unsubscribeRequests <-chan int,
) {
	subscribers := map[int]chan giftRowData{}
	history := make([]giftRowData, 0, maxTableRows)
	nextSubscriberID := 1

	for {
		select {
		case <-ctx.Done():
			for id, stream := range subscribers {
				close(stream)
				delete(subscribers, id)
			}
			return

		case request := <-subscribeRequests:
			stream := make(chan giftRowData, clientBufferSize)
			historyCopy := append([]giftRowData(nil), history...)
			request.response <- subscription{
				id:      nextSubscriberID,
				stream:  stream,
				history: historyCopy,
			}
			subscribers[nextSubscriberID] = stream
			nextSubscriberID++

		case subscriberID := <-unsubscribeRequests:
			stream, ok := subscribers[subscriberID]
			if !ok {
				continue
			}
			close(stream)
			delete(subscribers, subscriberID)

		case rawMessage := <-inbound:
			gift, ok, err := processReceivedMessage(rawMessage)
			if err != nil {
				log.Printf("drop websocket message: %v", err)
				continue
			}
			if !ok {
				continue
			}

			gift = transformGiftMessage(gift)
			row := buildGiftRowData(roomID, gift, giftCatalog)
			if persistGift != nil {
				persistGift(roomID, row)
			}
			history = append(history, row)
			if len(history) > maxTableRows {
				history = history[len(history)-maxTableRows:]
			}

			for subscriberID, stream := range subscribers {
				if enqueueLatest(stream, row) {
					continue
				}
				close(stream)
				delete(subscribers, subscriberID)
				log.Printf("subscriber %d dropped due to overflow", subscriberID)
			}
		}
	}
}

func processReceivedMessage(message []byte) (GiftMessage, bool, error) {
	payload, ok := extractJSONPayload(message)
	if !ok {
		return GiftMessage{}, false, nil
	}

	var envelope baseMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return GiftMessage{}, false, err
	}
	if envelope.Type != 2 {
		return GiftMessage{}, false, nil
	}

	var gift GiftMessage
	if err := json.Unmarshal(payload, &gift); err != nil {
		return GiftMessage{}, false, err
	}

	return gift, true, nil
}

func extractJSONPayload(message []byte) ([]byte, bool) {
	messageText := string(message)

	switch messageText {
	case "ACK\tshowroom", "Could not decode a text frame as UTF-8.":
		return nil, false
	}

	if !strings.HasPrefix(messageText, "MSG\t") {
		return nil, false
	}

	parts := strings.SplitN(messageText, "\t", 3)
	if len(parts) != 3 {
		return nil, false
	}
	payload := parts[2]
	return []byte(payload), true
}

func transformGiftMessage(gift GiftMessage) GiftMessage {
	return gift
}

func enqueueLatest(stream chan giftRowData, gift giftRowData) bool {
	select {
	case stream <- gift:
		return true
	default:
	}

	select {
	case <-stream:
	default:
	}

	select {
	case stream <- gift:
		return true
	default:
		return false
	}
}

package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
)

func runGiftHub(
	ctx context.Context,
	bcsvrkey string,
	inbound <-chan []byte,
	subscribeRequests <-chan subscriptionRequest,
	unsubscribeRequests <-chan int,
) {
	subscribers := map[int]chan GiftMessage{}
	history := make([]GiftMessage, 0, maxTableRows)
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
			stream := make(chan GiftMessage, clientBufferSize)
			historyCopy := append([]GiftMessage(nil), history...)
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
			gift, ok, err := processReceivedMessage(bcsvrkey, rawMessage)
			if err != nil {
				log.Printf("drop websocket message: %v", err)
				continue
			}
			if !ok {
				continue
			}

			gift = transformGiftMessage(gift)
			history = append(history, gift)
			if len(history) > maxTableRows {
				history = history[len(history)-maxTableRows:]
			}

			for subscriberID, stream := range subscribers {
				if enqueueLatest(stream, gift) {
					continue
				}
				close(stream)
				delete(subscribers, subscriberID)
				log.Printf("subscriber %d dropped due to overflow", subscriberID)
			}
		}
	}
}

func processReceivedMessage(bcsvrkey string, message []byte) (GiftMessage, bool, error) {
	payload, ok := extractJSONPayload(bcsvrkey, message)
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

func extractJSONPayload(bcsvrkey string, message []byte) ([]byte, bool) {
	messageText := string(message)

	switch messageText {
	case "ACK\tshowroom", "Could not decode a text frame as UTF-8.":
		return nil, false
	}

	prefix := "MSG\t" + bcsvrkey
	if !strings.HasPrefix(messageText, prefix) {
		return nil, false
	}

	payload := strings.TrimPrefix(messageText, prefix)
	payload = strings.TrimLeft(payload, "\t")
	return []byte(payload), true
}

func transformGiftMessage(gift GiftMessage) GiftMessage {
	return gift
}

func enqueueLatest(stream chan GiftMessage, gift GiftMessage) bool {
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

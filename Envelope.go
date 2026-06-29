package main

import "time"

const (
	maxTableRows     = 16
	clientBufferSize = 256
)

type baseMessage struct {
	Type int `json:"t"`
}

type GiftMessage struct {
	Type      int    `json:"t"`
	UserID    int    `json:"u"`
	Nickname  string `json:"ac"`
	Avatar    int    `json:"av"`
	GiftCode  int    `json:"g"`
	GiftType  int    `json:"gt"`
	Count     int    `json:"n"`
	H         int    `json:"h"`
	D         int    `json:"d"`
	At        int    `json:"at"`
	Ua        int    `json:"ua"`
	Aft       int    `json:"aft"`
	CreatedAt int64  `json:"created_at"`
	Cl        int    `json:"cl"`
	Cifn      string `json:"cifn"`
	Cbisc     string `json:"cbisc"`
	Cbiec     string `json:"cbiec"`
}

func (g GiftMessage) CreatedAtTime() time.Time {
	return time.Unix(g.CreatedAt, 0)
}

type subscription struct {
	id      int
	stream  <-chan giftRowData
	history []giftRowData
}

type subscriptionRequest struct {
	response chan subscription
}

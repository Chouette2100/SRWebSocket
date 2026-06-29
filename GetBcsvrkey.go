// Copyright © 2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	// "encoding/json"
	// "fmt"
	"fmt"
	// "net/url"
	"net/http"
	// "os"
	// "os/signal"
	// "strings"
	// "time"

	// "github.com/gorilla/websocket" // WebSocketクライアント/サーバーライブラリ

	"github.com/Chouette2100/srapi/v2"
)

type GiftCatalogItem struct {
	GiftName string
	Point    int
	Free     bool
}

func GetBcsvrkey(roomid int) (bcsvrkey string, err error) {
	lol, liveErr := srapi.ApiLiveOnlives3(http.DefaultClient)
	if liveErr == nil {
		if lol == nil {
			liveErr = fmt.Errorf("api returned nil live list")
		} else {
			for _, onlive := range lol.Onlives {
				for _, live := range onlive.Lives {
					if live.RoomID == roomid {
						bcsvrkey = live.BcsvrKey
						return
					}
				}
			}
			liveErr = fmt.Errorf("bcsvrkey not found in onlives for roomid=%d", roomid)
		}
	}

	usr, err := getUserInfo(roomid)
	if err != nil {
		return "", fmt.Errorf("get user info roomid=%d: %w", roomid, err)
	}

	status, err := srapi.ApiRoomStatus(http.DefaultClient, usr.Userid)
	if err != nil {
		return "", fmt.Errorf("room status roomid=%d userid=%s: %w (onlives err: %v)", roomid, usr.Userid, err, liveErr)
	}
	if status == nil {
		return "", fmt.Errorf("room status returned nil for roomid=%d (onlives err: %v)", roomid, liveErr)
	}
	if status.Broadcast_key != "" {
		return status.Broadcast_key, nil
	}

	return "", fmt.Errorf("bcsvrkey not found for roomid=%d userid=%s (onlives err: %v)", roomid, usr.Userid, liveErr)
}

func GetGiftCatalog(roomid int) (map[int]GiftCatalogItem, error) {
	lgl, err := srapi.ApiLiveGiftlist(http.DefaultClient, roomid)
	if err != nil {
		return nil, err
	}

	result := make(map[int]GiftCatalogItem)
	for _, gift := range lgl.Normal {
		result[gift.GiftID] = GiftCatalogItem{
			GiftName: gift.GiftName,
			Point:    gift.Point,
			Free:     gift.Free,
		}
	}
	for _, gift := range lgl.Enquete {
		result[gift.GiftID] = GiftCatalogItem{
			GiftName: gift.GiftName,
			Point:    gift.Point,
			Free:     gift.Free,
		}
	}

	return result, nil
}

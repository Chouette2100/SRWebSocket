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

func GetBcsvrkey(roomid int) (bcsvrkey string, err error) {

	lol, err := srapi.ApiLiveOnlives3(http.DefaultClient)

	for _, onlive := range lol.Onlives {
		for _, live := range onlive.Lives {
			if live.RoomID == roomid {
				bcsvrkey = live.BcsvrKey
				return
			}
		}
		if bcsvrkey == "" {
			err = fmt.Errorf("bcsvrkey not found for roomid=%d", roomid)
		}
	}
	return
}

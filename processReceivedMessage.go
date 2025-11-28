// Copyright © 2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	"encoding/json"
	// "fmt"
	"log"
	// "net/url"
	// "os"
	// "os/signal"
	"strings"
	// "time"
	// "github.com/gorilla/websocket" // WebSocketクライアント/サーバーライブラリ
)

// ----------------

type WsComment struct {
	T         int    `json:"t"`
	U         int    `json:"u"`
	Ac        string `json:"ac"`
	Av        int    `json:"av"`
	Cm        string `json:"cm"`
	D         int    `json:"d"`
	At        int    `json:"at"`
	Ua        int    `json:"ua"`
	Aft       int    `json:"aft"`
	CreatedAt int    `json:"created_at"`
}

// ----------------

type WsTelops struct {
	Telops        []Telops `json:"telops"`
	Telop         string   `json:"telop"`
	IsDisplayLogo int      `json:"is_display_logo"`
	Interval      int      `json:"interval"`
	T             int      `json:"t"`
	API           string   `json:"api"`
}
type Color struct {
	R int `json:"r"`
	B int `json:"b"`
	G int `json:"g"`
}
type Telops struct {
	Color  Color  `json:"color"`
	Text   string `json:"text"`
	LiveID string `json:"live_id"`
	Type   string `json:"type"`
}

// ----------------

type WsVisitMsg struct {
	T         int    `json:"t"`
	U         int    `json:"u"`
	Tt        int    `json:"tt"`
	M         string `json:"m"`
	Me        string `json:"me"`
	C         string `json:"c"`
	CreatedAt int    `json:"created_at"`
}

// ----------------

type WsGift struct {
	T         int    `json:"t"`
	U         int    `json:"u"`
	Ac        string `json:"ac"`
	Av        int    `json:"av"`
	G         int    `json:"g"`
	Gt        int    `json:"gt"`
	N         int    `json:"n"`
	H         int    `json:"h"`
	D         int    `json:"d"`
	At        int    `json:"at"`
	Ua        int    `json:"ua"`
	Aft       int    `json:"aft"`
	CreatedAt int    `json:"created_at"`
}

// ----------------

type WsT struct {
	CreatedAt int `json:"created_at"`
	T         int `json:"t"`
}

// ----------------

type WsCPT struct {
	CreatedAt int `json:"created_at"`
	C         int `json:"c"`
	P         int `json:"p"`
	T         int `json:"t"`
}

// ----------------

// processReceivedMessage は受信したメッセージを処理する関数です。
// JavaScriptの onmessage イベント内のロジックに相当します。
func processReceivedMessage(message []byte) {
	msgStr := string(message) // バイトスライスを文字列に変換

	// JavaScriptの if (message.data === "ACK\tshowroom" || ...) に相当
	if msgStr == "ACK\tshowroom" || msgStr == "Could not decode a text frame as UTF-8." {
		log.Println("疎通確認又はエラー:", msgStr)
	} else {
		// JavaScriptの JSON.parse(message.data.replace("MSG\tbXXXXX:XXXXXXXX", "")) に相当
		prefix := "MSG\t" + bcsvrkey
		if strings.HasPrefix(msgStr, prefix) {
			// プレフィックスを削除してJSON文字列を抽出
			jsonStr := strings.TrimPrefix(msgStr, prefix)
			log.Println("Received raw JSON string:", jsonStr)

			var myMsg MyMessage // 定義した構造体のインスタンス
			// JSON文字列をGoの構造体にデコード (アンマーシャル)
			err := json.Unmarshal([]byte(jsonStr), &myMsg)
			if err != nil {
				log.Println("JSON unmarshal error:", err)
				// JSONとしてパースできない場合は、元のメッセージをそのまま表示するなど
				log.Println("Original message (not valid JSON after prefix removal):", msgStr)
			} else {
				// 成功した場合、構造体の内容を表示
				log.Printf("Received JSON object: %+v", myMsg)
				// ここで myMsg のデータを使って具体的なアプリケーションロジックを実装します。
				// 例: データベースへの保存、他のサービスへの通知など
			}
		} else {
			// 予期しない形式のメッセージが来た場合
			log.Println("Received message with unexpected format:", msgStr)
		}
	}
}

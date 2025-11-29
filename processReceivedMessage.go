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

type WsComment struct { // 10
	T         int    `json:"t"`          // "1" : コメント？
	U         int    `json:"u"`          // ルームID
	Ac        string `json:"ac"`         // ルーム名
	Av        int    `json:"av"`         // アバターID
	Cm        string `json:"cm"`         // コメント
	D         int    `json:"d"`          // 不明
	At        int    `json:"at"`         // 不明
	Ua        int    `json:"ua"`         // 不明
	Aft       int    `json:"aft"`        // 不明
	CreatedAt int    `json:"created_at"` // タイムスタンプ
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

type WsGift struct { // 14
	T         int    `json:"t"`          // "2" : ギフト？
	U         int    `json:"u"`          // ルームID
	Ac        string `json:"ac"`         // ルーム名
	Av        int    `json:"av"`         // アバターID
	G         int    `json:"g"`          // ギフトID
	Gt        int    `json:"gt"`         // ギフトタイプ？
	Gn        string `json:"gn"`         // 不明
	Gc        int    `json:"gc"`         // 不明
	N         int    `json:"n"`          // 個数
	H         int    `json:"h"`          // 不明
	At        int    `json:"at"`         // 不明
	Ua        int    `json:"ua"`         // ルーム種別
	Aft       int    `json:"aft"`        // 不明
	CreatedAt int    `json:"created_at"` // タイムスタンプ
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
func processReceivedMessage(bcsvrkey string, message []byte) {
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

			// 1. まず、JSONを map[string]json.RawMessage にデコードし、キーの数を取得します。
			//    json.RawMessage を使うことで、値のデコードは後回しにし、キーの数だけを効率的に取得できます。
			var rawMap map[string]json.RawMessage
			if err := json.Unmarshal([]byte(jsonStr), &rawMap); err != nil {
				// return nil, fmt.Errorf("failed to unmarshal to raw map: %w", err)
				log.Printf("failed to unmarshal to raw map: %s", err.Error())
				return
			}

			memberCount := len(rawMap)
			log.Printf("  Detected member count: %d\n", memberCount)

			// var myMsg MyMessage // 定義した構造体のインスタンス
			// JSON文字列をGoの構造体にデコード (アンマーシャル)
			// var intf interface{}
			// err := json.Unmarshal([]byte(jsonStr), &intf)
			// if err != nil {
			// 	log.Println("JSON unmarshal error:", err)
			// 	// JSONとしてパースできない場合は、元のメッセージをそのまま表示するなど
			// 	log.Println("Original message (not valid JSON after prefix removal):", msgStr)
			// } else {
			// // 成功した場合、構造体の内容を表示
			// log.Printf("Received JSON object: %+v", myMsg)
			// // ここで myMsg のデータを使って具体的なアプリケーションロジックを実装します。
			// // 例: データベースへの保存、他のサービスへの通知など
			// interface{} の型アサーションで具体的な型に変換して処理
			// 	switch v := intf.(type) {
			// 	case map[string]interface{}:
			// }
		} else {
			// 予期しない形式のメッセージが来た場合
			log.Println("Received message with unexpected format:", msgStr)
		}
	}
}

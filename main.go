// Copyright © 2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	// "encoding/json"
	// "fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/websocket" // WebSocketクライアント/サーバーライブラリ

	"github.com/Chouette2100/srcom"
)

/*
000000 2025-11-28 最初のバージョン
000100 2025-11-28 ルームIDで取得対象を指定する、JSONデコードを複数の構造体に対応させるための準備、ログファイル書式変更
000101 2025-12-05 srcom.CreateLogfile3を使用する
*/

const Version = "000101"

// MyMessage は受信するJSONデータの構造体を定義します。
// 実際のJSONデータに合わせてフィールドを調整してください。
type MyMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	// 必要に応じて他のフィールドを追加
	// 例:
	// ID      int    `json:"id"`
	// Content string `json:"content"`
}

func main() {

	logfile, err := srcom.CreateLogfile3(Version, time.Now().Format("150405"))
	if err != nil {
		panic("cannnot open logfile: " + err.Error())
	}
	defer logfile.Close()

	// 起動時の最初のパラメータをルートIDとして取得
	if len(os.Args) < 2 {
		log.Fatal("Usage: SRWebSocket <roomid>")
	}
	roomid, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid roomid: %s", os.Args[1])
	}

	// 0. bcsvrkeyの取得
	bcsvrkey := ""
	bcsvrkey, err = GetBcsvrkey(roomid)
	if err != nil {
		log.Fatalf("GetBcsvrkey error: %v", err)
	}
	log.Printf("roomid=%d bcsvrkey=%s", roomid, bcsvrkey)

	// 1. WebSocket URLの準備
	// JavaScriptの 'wss://xxx.com' に相当します。
	// 実際のホスト名に合わせて 'xxx.com' の部分を変更してください。
	u := url.URL{Scheme: "wss", Host: "online.showroom-live.com", Path: "/"}
	log.Printf("connecting to %s", u.String())

	// 2. WebSocketサーバーへの接続
	// websocket.DefaultDialer.Dial は、指定されたURLにWebSocket接続を試みます。
	// 成功すると *websocket.Conn オブジェクトが返されます。
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err) // 接続失敗時はプログラムを終了
	}
	defer conn.Close() // main関数終了時に接続を確実にクローズ

	// 3. 接続が開いたときの処理 (JavaScriptの socket.onopen に相当)
	// 接続が成功したら、すぐに購読メッセージを送信します。
	log.Println("WebSocket connection opened. Sending subscription message.")
	err = conn.WriteMessage(websocket.TextMessage, []byte("SUB\t"+bcsvrkey))
	if err != nil {
		log.Println("write:", err)
		return
	}

	// 4. メッセージ受信ループをゴルーチンで実行 (JavaScriptの socket.onmessage に相当)
	// メッセージの受信はブロッキング処理なので、メインの処理を妨げないようゴルーチンで実行します。
	done := make(chan struct{}) // 受信ゴルーチンの終了を通知するためのチャネル
	go func() {
		defer close(done)
		for {
			// ReadMessage は次のメッセージを受信するまでブロックします。
			// messageType: テキストメッセージかバイナリメッセージかなど
			// message: 受信したデータ (バイトスライス)
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				// 接続がクローズされた場合やエラーが発生した場合
				log.Println("read error:", err)
				return // ゴルーチンを終了
			}

			// テキストメッセージのみを処理 (通常WebSocketはテキストかバイナリ)
			if messageType == websocket.TextMessage {
				processReceivedMessage(bcsvrkey, message)
			} else {
				log.Printf("Received non-text message type: %d, data: %s", messageType, message)
			}
		}
	}()

	// 5. 割り込みシグナル (Ctrl+Cなど) を待機してクリーンシャットダウン
	// プログラムが突然終了するのではなく、WebSocket接続を適切にクローズするための処理です。
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt) // OSからの割り込みシグナルを捕捉

	select {
	case <-done:
		// 受信ゴルーチンが終了した場合 (通常はエラー発生時)
		log.Println("Receive goroutine finished.")
	case <-interrupt:
		// Ctrl+C などでプログラムが中断された場合
		log.Println("Interrupt signal received. Closing WebSocket connection...")

		// サーバーにクローズメッセージを送信し、接続を正常に閉じます。
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("write close error:", err)
			return
		}

		// サーバーからのクローズ応答を待つか、タイムアウトするまで待機
		select {
		case <-done:
			log.Println("Server acknowledged close.")
		case <-time.After(time.Second):
			log.Println("Server close acknowledgment timed out.")
		}
	}
	log.Println("Exiting application.")
}

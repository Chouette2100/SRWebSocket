// Copyright © 2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Chouette2100/srcom"
)

/*
000000 2025-11-28 最初のバージョン
000100 2025-11-28 ルームIDで取得対象を指定する、JSONデコードを複数の構造体に対応させるための準備、ログファイル書式変更
000101 2025-12-05 srcom.CreateLogfile3を使用する
000200 2026-06-22 SSE/HTMX プロトタイプを追加
000201 2026-06-25 累計値の算出・表示とそれに伴うレイアウトの変更を行う
000300 2026-06-27 常時起動を前提とした機能の再構成を行う
000400 2026-06-28 タイムアウト対策を行う
*/

const Version = "000400"

func main() {
	logfile, err := srcom.CreateLogfile3(Version, time.Now().Format("150405"))
	if err != nil {
		panic("cannot open logfile: " + err.Error())
	}
	defer logfile.Close()

	listenAddr := flag.String("listen", ":8080", "HTTP listen address")
	idleTimeout := flag.Duration("idle-timeout", 5*time.Minute, "idle timeout before stopping a room worker")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	roomManager := NewRoomManager(ctx, *idleTimeout)

	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           newHTTPHandler(roomManager),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		log.Printf("listening on http://127.0.0.1%s", *listenAddr)
		log.Printf("room worker idle timeout: %s", idleTimeout.String())
		serverErrCh <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("application stopped")
}

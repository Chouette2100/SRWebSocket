package main

import (
	"log"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/Chouette2100/srdblib/v3"
	"github.com/go-gorp/gorp/v3"
)

var db *sql.DB
var dbconfig *srdblib.DBConfig
var dbmap *gorp.DbMap

func init() {
	var err error

	// 暗号化された設定ファイルを指定して読み込み
	dbcfg, err := loadEncryptedConfig("DBconfig.enc.yaml")
	if err != nil {
		log.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
	}

	// 2. DB接続 (gorp)
	constring := dbcfg.Acct + ":" + dbcfg.Password
	constring += "@tcp(" + dbcfg.Host + ":" + dbcfg.Port + ")/" + dbcfg.Name
	db, err := sql.Open("mysql", constring)
	if err != nil {
		log.Fatal(err)
	}
	dbmap = &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "utf8mb4"}}
	dbmap.AddTableWithName(srdblib.User{}, "user").SetKeys(false, "Userno")
	if err := ensureRoomConfigTable(); err != nil {
		log.Fatalf("room config table initialization failed: %v", err)
	}
	if err := ensureRoomEventLogTable(); err != nil {
		log.Fatalf("room event log table initialization failed: %v", err)
	}

}

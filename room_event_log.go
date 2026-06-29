package main

import (
	"encoding/json"
	"fmt"
)

func ensureRoomEventLogTable() error {
	if dbmap == nil {
		return fmt.Errorf("dbmap is not initialized")
	}

	_, err := dbmap.Exec(`
CREATE TABLE IF NOT EXISTS room_gift_event (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    roomid BIGINT NOT NULL,
    gift_name VARCHAR(255) NOT NULL DEFAULT '',
    gift_point INT NOT NULL DEFAULT 0,
    gift_free TINYINT(1) NOT NULL DEFAULT 0,
    count INT NOT NULL DEFAULT 0,
    pt INT NOT NULL DEFAULT 0,
    sum INT NOT NULL DEFAULT 0,
    nickname VARCHAR(255) NOT NULL DEFAULT '',
    user_id BIGINT NOT NULL DEFAULT 0,
    avatar INT NOT NULL DEFAULT 0,
    gift_code INT NOT NULL DEFAULT 0,
    gift_type INT NOT NULL DEFAULT 0,
    h INT NOT NULL DEFAULT 0,
    d INT NOT NULL DEFAULT 0,
    at INT NOT NULL DEFAULT 0,
    ua INT NOT NULL DEFAULT 0,
    aft INT NOT NULL DEFAULT 0,
    created_at_display VARCHAR(19) NOT NULL DEFAULT '',
    cl INT NOT NULL DEFAULT 0,
    message_type INT NOT NULL DEFAULT 0,
    payload_json LONGTEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_roomid_created_at (roomid, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
`)
	if err != nil {
		return err
	}

	columns := []struct {
		name string
		ddl  string
	}{
		{name: "gift_name", ddl: "VARCHAR(255) NOT NULL DEFAULT ''"},
		{name: "gift_point", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "gift_free", ddl: "TINYINT(1) NOT NULL DEFAULT 0"},
		{name: "count", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "pt", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "sum", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "nickname", ddl: "VARCHAR(255) NOT NULL DEFAULT ''"},
		{name: "user_id", ddl: "BIGINT NOT NULL DEFAULT 0"},
		{name: "avatar", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "gift_code", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "gift_type", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "h", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "d", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "at", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "ua", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "aft", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "created_at_display", ddl: "VARCHAR(19) NOT NULL DEFAULT ''"},
		{name: "cl", ddl: "INT NOT NULL DEFAULT 0"},
		{name: "message_type", ddl: "INT NOT NULL DEFAULT 0"},
	}

	for _, column := range columns {
		var exists int
		if err := dbmap.Db.QueryRow(`
SELECT COUNT(*)
  FROM information_schema.COLUMNS
 WHERE TABLE_SCHEMA = DATABASE()
   AND TABLE_NAME = 'room_gift_event'
   AND COLUMN_NAME = ?
`, column.name).Scan(&exists); err != nil {
			return fmt.Errorf("check column %s: %w", column.name, err)
		}
		if exists == 0 {
			if _, err := dbmap.Exec(fmt.Sprintf(`ALTER TABLE room_gift_event ADD COLUMN %s %s`, column.name, column.ddl)); err != nil {
				return fmt.Errorf("add column %s: %w", column.name, err)
			}
		}
	}

	return nil
}

func SaveGiftEvent(roomID int, row giftRowData) error {
	if dbmap == nil {
		return fmt.Errorf("dbmap is not initialized")
	}

	payload, err := json.Marshal(row.GiftMessage)
	if err != nil {
		return fmt.Errorf("marshal gift event: %w", err)
	}

	_, err = dbmap.Exec(`
INSERT INTO room_gift_event (
    roomid,
    gift_name,
    gift_point,
    gift_free,
    count,
    pt,
    sum,
    nickname,
    user_id,
    avatar,
    gift_code,
    gift_type,
    h,
    d,
    at,
    ua,
    aft,
    created_at_display,
    cl,
    message_type,
    payload_json,
    created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, FROM_UNIXTIME(?))
`, roomID,
		row.GiftName,
		row.GiftPoint,
		row.GiftFree,
		row.Count,
		row.Pt,
		row.Sum,
		row.Nickname,
		row.UserID,
		row.Avatar,
		row.GiftCode,
		row.GiftType,
		row.H,
		row.D,
		row.At,
		row.Ua,
		row.Aft,
		row.CreatedAtDisplay,
		row.Cl,
		row.Type,
		string(payload),
		row.CreatedAt,
	)
	return err
}

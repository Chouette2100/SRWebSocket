package main

import (
	"fmt"
)

type RoomRunMode string

const (
	RoomRunModeOnce   RoomRunMode = "once"
	RoomRunModeAlways RoomRunMode = "always"
)

type RoomConfig struct {
	RoomID    int         `db:"roomid" json:"roomid"`
	Mode      RoomRunMode `db:"mode" json:"mode"`
	SaveData  bool        `db:"save_data" json:"save_data"`
	CreatedAt string      `db:"created_at" json:"created_at"`
	UpdatedAt string      `db:"updated_at" json:"updated_at"`
}

func (c RoomConfig) Normalize() RoomConfig {
	if !c.Mode.Valid() {
		c.Mode = RoomRunModeOnce
	}
	return c
}

func (c RoomConfig) Validate() error {
	if c.RoomID <= 0 {
		return fmt.Errorf("invalid roomid: %d", c.RoomID)
	}
	if !c.Mode.Valid() {
		return fmt.Errorf("invalid room mode: %q", c.Mode)
	}
	return nil
}

func (m RoomRunMode) Valid() bool {
	switch m {
	case RoomRunModeOnce, RoomRunModeAlways:
		return true
	default:
		return false
	}
}

func ensureRoomConfigTable() error {
	if dbmap == nil {
		return fmt.Errorf("dbmap is not initialized")
	}

	_, err := dbmap.Exec(`
CREATE TABLE IF NOT EXISTS room_config (
    roomid BIGINT NOT NULL PRIMARY KEY,
    mode VARCHAR(16) NOT NULL,
    save_data TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
`)
	return err
}

func LoadRoomConfigs() ([]RoomConfig, error) {
	if dbmap == nil {
		return nil, fmt.Errorf("dbmap is not initialized")
	}

	var configs []RoomConfig
	_, err := dbmap.Select(&configs, `
SELECT roomid, mode, save_data,
       DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
       DATE_FORMAT(updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
FROM room_config
ORDER BY roomid
`)
	if err != nil {
		return nil, err
	}

	for i := range configs {
		configs[i] = configs[i].Normalize()
	}
	return configs, nil
}

func LoadRoomConfig(roomID int) (RoomConfig, error) {
	if dbmap == nil {
		return RoomConfig{}, fmt.Errorf("dbmap is not initialized")
	}

	var config RoomConfig
	err := dbmap.SelectOne(&config, `
SELECT roomid, mode, save_data,
       DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
       DATE_FORMAT(updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
FROM room_config
WHERE roomid = ?
`, roomID)
	if err != nil {
		return RoomConfig{}, err
	}
	return config.Normalize(), nil
}

func SaveRoomConfig(config RoomConfig) error {
	if dbmap == nil {
		return fmt.Errorf("dbmap is not initialized")
	}

	config = config.Normalize()
	if err := config.Validate(); err != nil {
		return err
	}

	_, err := dbmap.Exec(`
INSERT INTO room_config (roomid, mode, save_data, created_at, updated_at)
VALUES (?, ?, ?, NOW(), NOW())
ON DUPLICATE KEY UPDATE
    mode = VALUES(mode),
    save_data = VALUES(save_data),
    updated_at = NOW()
`, config.RoomID, string(config.Mode), config.SaveData)
	return err
}

func DeleteRoomConfig(roomID int) error {
	if dbmap == nil {
		return fmt.Errorf("dbmap is not initialized")
	}
	_, err := dbmap.Exec(`DELETE FROM room_config WHERE roomid = ?`, roomID)
	return err
}

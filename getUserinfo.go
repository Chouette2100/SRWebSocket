package main

import (
	"fmt"
	"log"

	"github.com/Chouette2100/srdblib/v3"
)

var selectOneUserInfo = func(dest interface{}, stmt string, args ...interface{}) error {
	if dbmap == nil {
		return fmt.Errorf("dbmap is not initialized")
	}
	return dbmap.SelectOne(dest, stmt, args...)
}

func getUserInfo(userno int) (*srdblib.User, error) {
	user := srdblib.User{}
	stmt := "SELECT userno, userid, user_name FROM user WHERE userno = ?"
	err := selectOneUserInfo(&user, stmt, userno)
	if err != nil {
		log.Printf("Error retrieving user info for userno %d: %v", userno, err)
		return nil, err
	}
	return &user, nil
}

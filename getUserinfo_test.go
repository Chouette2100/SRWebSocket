package main

import (
	"reflect"
	"testing"

	"github.com/Chouette2100/srdblib/v3"
)

func Test_getUserInfo(t *testing.T) {

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		userno  int
		want    *srdblib.User
		wantErr bool
	}{
		{
			name:   "Valid userno",
			userno: 130997,
			want: &srdblib.User{
				Userno:    130997,
				Userid:    "615191698244",
				User_name: "stubbed user name",
			},
			wantErr: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldSelectOneUserInfo := selectOneUserInfo
			defer func() { selectOneUserInfo = oldSelectOneUserInfo }()
			selectOneUserInfo = func(dest interface{}, stmt string, args ...interface{}) error {
				user, ok := dest.(*srdblib.User)
				if !ok {
					t.Fatalf("unexpected destination type %T", dest)
				}
				if stmt != "SELECT userno, userid, user_name FROM user WHERE userno = ?" {
					t.Fatalf("unexpected stmt: %s", stmt)
				}
				if len(args) != 1 {
					t.Fatalf("unexpected args length: %d", len(args))
				}
				user.Userno = args[0].(int)
				user.Userid = "615191698244"
				user.User_name = "stubbed user name"
				return nil
			}

			got, gotErr := getUserInfo(tt.userno)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("getUserInfo() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("getUserInfo() succeeded unexpectedly")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUserInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

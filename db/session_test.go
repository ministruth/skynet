package db

import (
	"bytes"
	"encoding/gob"
	"testing"
	"time"

	"github.com/MXWXZ/skynet/sn"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
)

func TestLoadSessionBytes(t *testing.T) {
	var valid bytes.Buffer
	test_obj := make(map[any]any)
	test_obj["id"] = uuid.New()
	test_obj["time"] = time.Now().Unix()
	enc := gob.NewEncoder(&valid)
	if err := enc.Encode(test_obj); err != nil {
		panic(err)
	}

	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *sn.SessionData
		wantErr bool
	}{
		{
			"Valid session string",
			args{valid.Bytes()},
			&sn.SessionData{
				ID:   test_obj["id"].(uuid.UUID),
				Time: test_obj["time"].(int64),
			},
			false,
		},
		{
			"Invalid session string",
			args{[]byte("abc")},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadSessionBytes(tt.args.b)
			assert.Equal(t, tt.wantErr, err != nil, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadSession(t *testing.T) {
	valid_want := sn.SessionData{ID: uuid.New(), Time: time.Now().Unix()}
	valid := sessions.Session{}
	valid.Values = make(map[interface{}]interface{})
	valid.Values["id"] = valid_want.ID
	valid.Values["time"] = valid_want.Time

	invalid := sessions.Session{}
	invalid.Values = make(map[interface{}]interface{})
	invalid.Values["ID"] = valid_want.ID
	invalid.Values["time"] = valid_want.Time

	type args struct {
		session *sessions.Session
	}
	tests := []struct {
		name    string
		args    args
		want    *sn.SessionData
		wantErr bool
	}{
		{"Valid session", args{&valid}, &valid_want, false},
		{"Invalid session", args{&invalid}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadSession(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

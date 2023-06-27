package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/MXWXZ/skynet/security"
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setup() {
	if err := newInstance(); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func TestToken(t *testing.T) {
	post := func(tt *testing.T, code int, header string, cookie *http.Cookie) error {
		var reqBody io.Reader
		tmp, err := json.Marshal(mss{
			"username": rootUser,
			"password": rootPass,
		})
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(tmp)
		req, err := http.NewRequest("POST",
			"http://"+viper.GetString("listen.address")+"/api/signin",
			reqBody)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		if header != "" {
			req.Header.Add(security.CSRF_HEADER, header)
		}
		c := &http.Client{}
		rsp, err := c.Do(req)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()
		body, err := io.ReadAll(rsp.Body)
		if err != nil {
			panic(err)
		}
		assert := assert.New(tt)
		if assert.Equal(code, rsp.StatusCode) && code == 200 {
			var rspData map[string]any
			if err := json.Unmarshal(body, &rspData); err != nil {
				return err
			}
			if assert.Contains(rspData, "code") {
				assert.EqualValues(0, rspData["code"])
			}
		}
		return nil
	}

	t.Run("No header/cookie POST", func(t *testing.T) {
		err := post(t, 400, "", nil)
		assert.Nil(t, err)
	})
	t.Run("No header POST", func(t *testing.T) {
		err := post(t, 400, "", getToken(t))
		assert.Nil(t, err)
	})
	t.Run("No cookie POST", func(t *testing.T) {
		token := getToken(t)
		err := post(t, 400, token.Value, nil)
		assert.Nil(t, err)
	})
	t.Run("Mismatch header/cookie POST_1", func(t *testing.T) {
		token := getToken(t)
		err := post(t, 400, utils.RandString(32), token)
		assert.Nil(t, err)
	})
	t.Run("Mismatch header/cookie POST_2", func(t *testing.T) {
		cookie := getToken(t)
		token := cookie.Value
		cookie.Value = utils.RandString(32)
		err := post(t, 400, token, cookie)
		assert.Nil(t, err)
	})
	t.Run("Wrong header/cookie POST", func(t *testing.T) {
		token := getToken(t)
		token.Value = utils.RandString(32)
		err := post(t, 400, token.Value, token)
		assert.Nil(t, err)
	})
	t.Run("Success POST", func(t *testing.T) {
		token := getToken(t)
		err := post(t, 200, token.Value, token)
		assert.Nil(t, err)
	})
}

func TestPing(t *testing.T) {
	t.Run("Ping Success", func(t *testing.T) {
		err := test(&testCase{
			url: "/ping",
		}, t)
		assert.Nil(t, err)
	})
}

func TestAuth(t *testing.T) {
	tests := []testCase{
		{
			name:   "Signin Fail All",
			url:    "/signin",
			method: "POST",
			body: mss{
				"username": "no",
				"password": "no",
			},
			code: sn.CodeUserInvalid,
		},
		{
			name:   "Signin Fail User",
			url:    "/signin",
			method: "POST",
			body: mss{
				"username": "no",
				"password": rootPass,
			},
			code: sn.CodeUserInvalid,
		},
		{
			name:   "Signin Fail Pass",
			url:    "/signin",
			method: "POST",
			body: mss{
				"username": rootUser,
				"password": "no",
			},
			code: sn.CodeUserInvalid,
		},
		{
			name:   "Signin Success",
			url:    "/signin",
			method: "POST",
			body: mss{
				"username": rootUser,
				"password": rootPass,
			},
		},
		{
			name:     "Signout Fail",
			url:      "/signout",
			method:   "POST",
			httpCode: 403,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test(t)
			assert.Nil(t, err)
		})
	}
	t.Run("Signout Success", func(t *testing.T) {
		c := loginRoot(t)
		err := test(&testCase{
			url:    "/signout",
			method: "POST",
			client: c,
		}, t)
		assert.Nil(t, err)
	})
}

func TestAccess(t *testing.T) {
	tests := []testCase{
		{
			name: "Access Guest",
			url:  "/access",
			data: msa{
				"signin":     false,
				"permission": map[string]any{"guest": sn.PermAll},
			},
		},
		{
			name: "Access User",
			url:  "/access",
			data: msa{
				"signin":     true,
				"id":         normalID,
				"permission": map[string]any{"guest": sn.PermAll, "user": sn.PermAll},
			},
			client: loginNormal(t),
		},
		{
			name: "Access Root",
			url:  "/access",
			data: msa{
				"signin":     true,
				"id":         rootID,
				"permission": map[string]any{"all": 7, "guest": sn.PermAll, "user": sn.PermAll},
			},
			client: loginRoot(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test(t)
			assert.Nil(t, err)
		})
	}
}

package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"reflect"
	"skynet/api"
	"skynet/cmd"
	"skynet/db"
	"skynet/handler"
	"skynet/security"
	"skynet/sn"
	"skynet/utils"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type msa = map[string]any
type mss = map[string]string

var (
	rootUser   string
	rootPass   string
	normalUser string
	normalPass string
)

func checkAddress(address string) bool {
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil && !os.IsTimeout(err) {
		return true
	}
	conn.Close()
	return false
}

func newInstance() (err error) {
	for {
		port := 10000 + rand.Intn(50000)
		addr := fmt.Sprintf("127.0.0.1:%v", port)
		if checkAddress(addr) {
			viper.Set("listen.address", addr)
			go cmd.Execute([]string{"run"})
			break
		}
	}
	for {
		if sn.Running {
			break
		}
	}

	rootUser = "test_" + utils.RandString(6)
	normalUser = "test_" + utils.RandString(6)
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var u *db.User
		u, rootPass, err = handler.User.WithTx(tx).New(rootUser, "", []byte{0})
		if err != nil {
			return err
		}
		_, err = handler.Group.WithTx(tx).Link([]uuid.UUID{u.ID}, []uuid.UUID{db.GetDefaultID(db.GroupRootID)})
		return err
	})
	_, normalPass, err = handler.User.New(normalUser, "", []byte{0})
	if err != nil {
		return err
	}
	return
}

type customType interface {
	test(t *testing.T, obj any)
}

type typeAnyString struct {
	min int
	max int
}

func (self *typeAnyString) test(t *testing.T, s any) {
	switch v := s.(type) {
	case string:
		if self.min == 0 && self.max == 0 {
			return
		}
		assert.GreaterOrEqual(t, len(v), self.min)
		assert.LessOrEqual(t, len(v), self.max)
	default:
		assert.FailNow(t, "data type is not string")
	}
}

func anyString() *typeAnyString {
	return &typeAnyString{}
}

func anyStringLen(l int) *typeAnyString {
	return &typeAnyString{min: l, max: l}
}

type testCase struct {
	name   string
	url    string
	method string
	body   map[string]string
	client *http.Client

	httpCode int
	code     api.RspCode
	data     any
}

func getToken(t *testing.T) string {
	rsp, err := http.Get("http://" + viper.GetString("listen.address") + "/api/token")
	assert.Nil(t, err)
	defer rsp.Body.Close()
	body, err := io.ReadAll(rsp.Body)
	var rspData map[string]any
	err = json.Unmarshal(body, &rspData)
	assert.Nil(t, err)
	if assert.Contains(t, rspData, "data") {
		switch v := rspData["data"].(type) {
		case string:
			t.Log("Get token", v)
			return v
		default:
			assert.FailNow(t, "data type is not string, but "+reflect.TypeOf(v).String())
		}
	}
	panic("Failed to get token")
}

func loginNormal(t *testing.T) *http.Client {
	return login(t, normalUser, normalPass)
}

func loginRoot(t *testing.T) *http.Client {
	return login(t, rootUser, rootPass)
}

func login(t *testing.T, user string, pass string) *http.Client {
	ret := &http.Client{}
	test(&testCase{
		url:    "/signin",
		method: "POST",
		body: mss{
			"username": user,
			"password": pass,
		},
		client: ret,
	}, t)
	return ret
}

func test(c *testCase, t *testing.T) error {
	return c.test(t)
}

func (c *testCase) test(t *testing.T) error {
	if c.httpCode == 0 {
		c.httpCode = 200
	}
	if c.method == "" {
		c.method = "GET"
	}
	if c.client == nil {
		c.client = &http.Client{}
	}

	var reqBody io.Reader
	if c.body != nil {
		tmp, err := json.Marshal(c.body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(tmp)
	}

	req, err := http.NewRequest(c.method,
		"http://"+viper.GetString("listen.address")+"/api"+c.url,
		reqBody)
	if err != nil {
		return err
	}
	if c.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.method != "GET" {
		req.Header.Add(security.CSRFHeader, getToken(t))
	}
	rsp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if c.client.Jar == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			panic(err)
		}
		c.client.Jar = jar
	}
	c.client.Jar.SetCookies(rsp.Request.URL, rsp.Cookies())
	defer rsp.Body.Close()
	body, err := io.ReadAll(rsp.Body)

	t.Log("Body:", string(body))

	assert := assert.New(t)
	if assert.Equal(c.httpCode, rsp.StatusCode) && c.httpCode == 200 {
		var rspData map[string]any
		if err := json.Unmarshal(body, &rspData); err != nil {
			return err
		}
		if assert.Contains(rspData, "code") {
			assert.EqualValues(c.code, rspData["code"])
		}
		if c.data != nil {
			if assert.Contains(rspData, "data") {
				switch v := c.data.(type) {
				case customType:
					v.test(t, rspData["data"])
				default:
					a, err := json.Marshal(v)
					if err != nil {
						return err
					}
					b, err := json.Marshal(rspData["data"])
					if err != nil {
						return err
					}
					assert.JSONEq(string(a), string(b))
				}
			}
		} else {
			assert.NotContains(rspData, "data")
		}
	}
	return nil
}

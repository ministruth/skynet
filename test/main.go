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
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/MXWXZ/skynet/cmd"
	"github.com/MXWXZ/skynet/security"
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/translator"
	"github.com/MXWXZ/skynet/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type msa = map[string]any
type mss = map[string]string

var (
	rootID     uuid.UUID
	rootUser   string
	rootPass   string
	normalID   uuid.UUID
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
	ts := time.Now()
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
		if sn.Skynet.StartTime.After(ts) {
			time.Sleep(1 * time.Second)
			break
		}
	}

	rootUser = "test_" + utils.RandString(6)
	rootPass = utils.RandString(8)
	normalUser = "test_" + utils.RandString(6)
	normalPass = utils.RandString(8)
	err = sn.Skynet.DB.Transaction(func(tx *gorm.DB) error {
		var u *sn.User
		u, err = sn.Skynet.User.WithTx(tx).New(rootUser, rootPass, "")
		if err != nil {
			return err
		}
		rootID = u.ID
		_, err = sn.Skynet.Group.WithTx(tx).Link([]uuid.UUID{u.ID}, []uuid.UUID{sn.Skynet.ID.Get(sn.GroupRootID)})
		return err
	})
	u, err := sn.Skynet.User.New(normalUser, normalPass, "")
	if err != nil {
		return err
	}
	normalID = u.ID
	return
}

type testCase struct {
	name   string
	url    string
	method string
	body   map[string]string
	client *http.Client

	translator *i18n.Localizer
	httpCode   int
	code       sn.ResponseCode
	msg        string
	data       any
}

func getToken(t *testing.T) *http.Cookie {
	csrf := &testCase{
		url:    "/token",
		method: "GET",
	}
	csrf.test(t)
	u, err := url.Parse("http://" + viper.GetString("listen.address"))
	if err != nil {
		panic(err)
	}
	for _, v := range csrf.client.Jar.Cookies(u) {
		if v.Name == security.CSRF_COOKIE {
			return v
		}
	}
	panic("No csrf token")
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
	if c.translator == nil {
		c.translator = translator.NewLocalizer("en-US")
	}
	if c.msg == "" {
		c.msg = c.code.GetMsg()
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
		token := getToken(t)
		req.AddCookie(token)
		req.Header.Add(security.CSRF_HEADER, token.Value)
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
	if err != nil {
		panic(err)
	}

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
		if assert.Contains(rspData, "msg") {
			assert.EqualValues(translator.TranslateString(c.translator, c.msg), rspData["msg"])
		}
		if c.data != nil {
			if assert.Contains(rspData, "data") {
				a, err := json.Marshal(c.data)
				if err != nil {
					return err
				}
				b, err := json.Marshal(rspData["data"])
				if err != nil {
					return err
				}
				assert.JSONEq(string(a), string(b))
			}
		} else {
			assert.NotContains(rspData, "data")
		}
	}
	return nil
}

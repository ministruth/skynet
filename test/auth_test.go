package test

import (
	"os"
	"skynet/api"
	"skynet/sn"
	"testing"

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
	tests := []testCase{
		{
			name: "Get Single Token",
			url:  "/token",
			data: anyStringLen(32),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test(t)
			assert.Nil(t, err)
		})
	}
	t.Run("Get Multiple Token", func(t *testing.T) {
		a := getToken(t)
		b := getToken(t)
		assert.NotEqual(t, a, b)
	})
}

func TestPing(t *testing.T) {
	t.Run("Ping Success", func(t *testing.T) {
		err := test(&testCase{
			url: "/ping",
		}, t)
		assert.Nil(t, err)

		sn.Running = false
		err = test(&testCase{
			url:  "/ping",
			code: api.CodeRestarting,
		}, t)
		assert.Nil(t, err)
		sn.Running = true
	})
}

func TestRestart(t *testing.T) {
	t.Run("Restart Fail", func(t *testing.T) {
		err := test(&testCase{
			url:      "/reload",
			method:   "POST",
			httpCode: 403,
		}, t)
		assert.Nil(t, err)

		c := loginNormal(t)
		err = test(&testCase{
			url:      "/reload",
			method:   "POST",
			httpCode: 403,
			client:   c,
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
			code: api.CodeInvalidUserOrPass,
		},
		{
			name:   "Signin Fail User",
			url:    "/signin",
			method: "POST",
			body: mss{
				"username": "no",
				"password": rootPass,
			},
			code: api.CodeInvalidUserOrPass,
		},
		{
			name:   "Signin Fail Pass",
			url:    "/signin",
			method: "POST",
			body: mss{
				"username": rootUser,
				"password": "no",
			},
			code: api.CodeInvalidUserOrPass,
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
				"permission": make(map[string]any),
			},
		},
		{
			name: "Access User",
			url:  "/access",
			data: msa{
				"signin":     true,
				"permission": make(map[string]any),
			},
			client: loginNormal(t),
		},
		{
			name: "Access Root",
			url:  "/access",
			data: msa{
				"signin":     true,
				"permission": map[string]any{"all": 7},
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

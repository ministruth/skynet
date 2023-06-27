package api

import (
	"github.com/MXWXZ/skynet/config"
	"github.com/MXWXZ/skynet/sn"
	"github.com/spf13/viper"
)

func APIGetPublicSetting(req *sn.Request) (*sn.Response, error) {
	ret := make(map[string]interface{})
	for _, v := range config.DefaultSetting {
		if v.Public {
			ret[v.Name] = viper.Get(v.Name)
		}
	}
	return &sn.Response{Data: ret}, nil
}

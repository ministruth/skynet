package api

import (
	"skynet/config"
)

func APIGetPublicSetting(req *Request) (*Response, error) {
	ret := make(map[string]interface{})
	for _, v := range config.DefaultSetting {
		if v.Public {
			ret[v.Name] = v.Value
		}
	}
	return &Response{Data: ret}, nil
}

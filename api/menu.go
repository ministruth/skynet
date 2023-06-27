package api

import (
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/translator"
)

func APIGetMenu(req *sn.Request) (*sn.Response, error) {
	type Ret struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Icon     string `json:"icon"`
		Badge    int64  `json:"badge"`
		Children []*Ret `json:"children"`
	}
	var dfs func([]*sn.MenuItem) []*Ret
	dfs = func(base []*sn.MenuItem) []*Ret {
		ret := []*Ret{}
		for _, v := range base {
			if v.Check(req.Perm) {
				ele := &Ret{
					Name: translator.TranslateString(req.Translator, v.Name),
					Icon: v.Icon,
					Path: v.Path,
				}
				if v.BadgeFunc != nil {
					ele.Badge = v.BadgeFunc()
				}
				ele.Children = dfs(v.Children)
				if !(len(ele.Children) == 0 && ele.Path == "" && v.OmitEmpty) { // hide empty menu group
					ret = append(ret, ele)
				}
			}
		}
		return ret
	}
	return &sn.Response{Data: dfs(sn.Skynet.Menu.GetAll())}, nil
}

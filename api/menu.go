package api

import (
	"skynet/translator"
	"skynet/utils/tpl"
	"strings"

	"github.com/google/uuid"
)

func GetMenuPluginID(req *Request) ([]uuid.UUID, error) {
	var dfs func([]*MenuItem) []uuid.UUID
	dfs = func(base []*MenuItem) []uuid.UUID {
		ret := []uuid.UUID{}
		for _, v := range base {
			if v.Check(req.Perm) {
				if strings.HasPrefix(v.Path, "/plugin/") {
					ids := strings.Split(v.Path, "/")
					if len(ids) >= 3 {
						id, err := uuid.Parse(ids[2])
						if err == nil {
							ret = append(ret, id)
						}
					}
				}
				ret = append(ret, dfs(v.Children)...)
			}
		}
		return ret
	}
	return tpl.NewSliceFinder(dfs(Menu)).GetSlice(), nil
}

func APIGetMenu(req *Request) (*Response, error) {
	type Ret struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Icon     string `json:"icon"`
		Children []*Ret `json:"children"`
	}
	var dfs func([]*MenuItem) []*Ret
	dfs = func(base []*MenuItem) []*Ret {
		ret := []*Ret{}
		for _, v := range base {
			if v.Check(req.Perm) {
				ele := &Ret{
					Name: translator.TranslateString(req.Translator, v.Name),
					Icon: v.Icon,
					Path: v.Path,
				}
				ele.Children = dfs(v.Children)
				if !(len(ele.Children) == 0 && ele.Path == "" && v.OmitEmpty) { // hide empty menu group
					ret = append(ret, ele)
				}
			}
		}
		return ret
	}
	return &Response{Data: dfs(Menu)}, nil
}

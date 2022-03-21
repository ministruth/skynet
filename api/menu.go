package api

import (
	"skynet/sn"
	"skynet/sn/impl"
	"skynet/sn/tpl"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func GetMenuPluginID(c *gin.Context) ([]uuid.UUID, error) {
	session, err := impl.GetSessionData(c)
	if err != nil {
		return []uuid.UUID{}, nil // not login
	}
	perm, err := impl.GetPerm(session.ID)
	if err != nil {
		return nil, err
	}
	var dfs func([]*sn.SNMenu) []uuid.UUID
	dfs = func(base []*sn.SNMenu) []uuid.UUID {
		ret := []uuid.UUID{}
		for _, v := range base {
			if (v.Checker == nil && (v.Perm == nil || v.Perm.ID == uuid.Nil)) ||
				(v.Checker == nil && impl.CheckPerm(perm, v.Perm)) ||
				(v.Checker != nil && v.Checker(perm)) {
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
	return tpl.NewSliceFinder(dfs(sn.Skynet.API.GetMenu())).GetSlice(), nil
}

func APIGetMenu(c *gin.Context, id uuid.UUID) (int, error) {
	type Ret struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Icon     string `json:"icon"`
		Children []*Ret `json:"children"`
	}
	perm, err := impl.GetPerm(id)
	if err != nil {
		return 500, err
	}
	var dfs func([]*sn.SNMenu) []*Ret
	translator := c.MustGet("translator").(*i18n.Localizer)
	dfs = func(base []*sn.SNMenu) []*Ret {
		var ret []*Ret
		for _, v := range base {
			if (v.Checker == nil && (v.Perm == nil || v.Perm.ID == uuid.Nil)) ||
				(v.Checker == nil && impl.CheckPerm(perm, v.Perm)) ||
				(v.Checker != nil && v.Checker(perm)) {
				ele := &Ret{
					Name: impl.TranslateString(translator, v.Name),
					Icon: v.Icon,
					Path: v.Path,
				}
				ele.Children = dfs(v.Children)
				if !(ele.Children == nil && ele.Path == "") { // hide empty menu group
					ret = append(ret, ele)
				}
			}
		}
		return ret
	}
	responseData(c, dfs(sn.Skynet.API.GetMenu()))
	return 0, nil
}

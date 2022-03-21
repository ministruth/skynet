package api

import (
	"errors"
	"net/http"
	"skynet/sn"
	"skynet/sn/impl"
	"skynet/sn/utils"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

/**
 * @apiDefine 400
 * @apiError (Error 4xx) 400 The request parameter is invalid.
 */

/**
 * @apiDefine 500
 * @apiError (Error 5xx) 500 Internal error occured.
 */

type Response struct {
	Code int32  `json:"code" example:"0"`
	Msg  string `json:"msg" example:"Success"`
	Data any    `json:"data" example:"[]"`
}

type siteAPI struct {
	router *gin.RouterGroup
	api    []*sn.SNAPIItem
	menu   []*sn.SNMenu
	lock   sync.Mutex
}

func (s *siteAPI) GetRouter() *gin.RouterGroup {
	return s.router
}

func (s *siteAPI) GetAPI() []*sn.SNAPIItem {
	return s.api
}

func (s *siteAPI) GetMenu() []*sn.SNMenu {
	return s.menu
}

func (s *siteAPI) CheckSignIn(c *gin.Context) (*impl.SessionData, error) {
	data, err := c.Cookie(viper.GetString("session.cookie"))
	if errors.Is(err, http.ErrNoCookie) {
		return nil, nil
	}
	if err != nil || data == "" {
		return nil, tracerr.Wrap(err)
	}
	session, err := impl.GetCTXSession(c)
	if err != nil {
		return nil, err
	}
	ret, err := impl.LoadSession(session)
	if err != nil {
		session.Options.MaxAge = -1 // delete invalid
		return nil, impl.SaveCTXSession(c)
	}
	c.Set("session", ret)
	return ret, nil
}

func (s *siteAPI) Handler(i *sn.SNAPIItem) func(c *gin.Context) {
	handler := func(c *gin.Context) (id uuid.UUID, code int, err error) {
		session, err := s.CheckSignIn(c)
		if err != nil {
			return uuid.Nil, 500, err
		}
		var perm map[uuid.UUID]*sn.SNPerm
		if session != nil {
			id = session.ID
			perm, err = impl.GetPerm(id)
			if err != nil {
				return id, 500, err
			}
		}
		if (i.Checker == nil && !impl.CheckPerm(perm, i.Perm)) ||
			(i.Checker != nil && !i.Checker(perm)) {
			return id, 403, nil
		}

		lang := c.DefaultQuery("lang", "en-US")
		c.Set("translator", i18n.NewLocalizer(sn.Skynet.Translator, lang))
		code, err = i.Func(c, id)
		return
	}
	return func(c *gin.Context) {
		id, code, err := handler(c)
		if err != nil {
			utils.WithLogTrace(wrap(c, id, nil), err).Error(err)
			c.AbortWithStatus(code)
			return
		}
		if code == 403 {
			c.String(403, "Permission denied")
			return
		} else if code != 0 && code != 200 {
			c.AbortWithStatus(code)
			return
		}
	}
}

func (s *siteAPI) AddAPI(item []*sn.SNAPIItem) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, v := range item {
		if v.Checker == nil && (v.Perm == nil || v.Perm.ID == uuid.Nil) {
			log.Fatalf("API %v do not have permission", v.Path)
		}
		fun := s.Handler(v)
		switch v.Method {
		case sn.APIGet:
			s.router.GET(v.Path, fun)
		case sn.APIPost:
			s.router.POST(v.Path, fun)
		case sn.APIPut:
			s.router.PUT(v.Path, fun)
		case sn.APIPatch:
			s.router.PATCH(v.Path, fun)
		case sn.APIDelete:
			s.router.DELETE(v.Path, fun)
		case sn.APIOptions:
			s.router.OPTIONS(v.Path, fun)
		case sn.APIHead:
			s.router.HEAD(v.Path, fun)
		case sn.APIAny:
			s.router.Any(v.Path, fun)
		}
	}
	s.api = append(s.api, item...)
}

func (s *siteAPI) getMenuByID(id uuid.UUID) *sn.SNMenu {
	var dfs func([]*sn.SNMenu) *sn.SNMenu
	dfs = func(this []*sn.SNMenu) *sn.SNMenu {
		for _, v := range this {
			if v.ID == id {
				return v
			}
			if ret := dfs(v.Children); ret != nil {
				return ret
			}
		}
		return nil
	}
	return dfs(s.menu)
}

func (s *siteAPI) AddMenu(item *sn.SNMenu, parent uuid.UUID) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	if parent == uuid.Nil {
		s.menu = append(s.menu, item)
		return true
	}

	p := s.getMenuByID(parent)
	if p == nil {
		return false
	}
	p.Children = append(p.Children, item)
	return true
}

// NewAPI returns new API object.
func NewAPI(r *gin.RouterGroup) sn.SNAPI {
	var ret siteAPI
	ret.router = r
	ret.AddAPI(initAPI())
	ret.lock.Lock()
	defer ret.lock.Unlock()
	ret.menu = initMenu()
	return &ret
}

type paginationParam struct {
	Page int `form:"page,default=1" binding:"min=1"`
	Size int `form:"size,default=10" binding:"min=1"`
}

type createdParam struct {
	CreatedSort  string `form:"createdSort" binding:"omitempty,oneof=asc desc"`
	CreatedStart int64  `form:"createdStart" binding:"min=0"`
	CreatedEnd   int64  `form:"createdEnd" binding:"min=0"`
}

type updatedParam struct {
	UpdatedSort  string `form:"updatedSort" binding:"omitempty,oneof=asc desc"`
	UpdatedStart int64  `form:"updatedStart" binding:"min=0"`
	UpdatedEnd   int64  `form:"updatedEnd" binding:"min=0"`
}

type idURI struct {
	ID string `uri:"id" binding:"required,uuid"`
}

func buildCondition(created *createdParam, updated *updatedParam,
	pageParam *paginationParam, text string, condText string) *sn.SNCondition {
	now := time.Now().UnixMilli()
	if created != nil && created.CreatedEnd == 0 {
		created.CreatedEnd = now
	}
	if updated != nil && updated.UpdatedEnd == 0 {
		updated.UpdatedEnd = now
	}

	cond := &sn.SNCondition{
		Limit:  pageParam.Size,
		Offset: (pageParam.Page - 1) * pageParam.Size,
	}

	if created != nil && !(created.CreatedStart == 0 && created.CreatedEnd == now) {
		cond.And("created_at BETWEEN ? AND ?", created.CreatedStart, created.CreatedEnd)
	}
	if updated != nil && !(updated.UpdatedStart == 0 && updated.UpdatedEnd == now) {
		cond.And("updated_at BETWEEN ? AND ?", updated.UpdatedStart, updated.UpdatedEnd)
	}
	if text != "" {
		cond.And(condText)
		count := strings.Count(condText, "?")
		for i := 0; i < count; i++ {
			cond.Args = append(cond.Args, "%"+text+"%")
		}
	}

	if updated != nil && updated.UpdatedSort != "" {
		cond.Order = []any{"updated_at " + updated.UpdatedSort}
	}
	if created != nil && created.CreatedSort != "" {
		cond.Order = []any{"created_at " + created.CreatedSort}
	}
	return cond
}

func wrap(c *gin.Context, u uuid.UUID, f log.Fields) *log.Entry {
	ret := log.WithFields(f)
	tmp := utils.MustMarshal(c.Request.URL.Query())
	if tmp != "{}" {
		ret = ret.WithField("param", tmp)
	}
	if u != uuid.Nil {
		ret = ret.WithField("user", u)
	}
	ret = ret.WithField("ip", utils.GetIP(c))
	return ret
}

func success(l *log.Entry, s string) {
	l.Info(s)
	sn.Skynet.Notification.New(sn.NotifySuccess, "Skynet log", s, utils.MustMarshal(l.Data))
}

package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/translator"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

type APIImpl struct {
	api []*sn.APIItem
}

func (impl *APIImpl) checkSignIn(c *gin.Context) (*sn.SessionData, error) {
	data, err := c.Cookie(viper.GetString("session.cookie"))
	if errors.Is(err, http.ErrNoCookie) {
		return nil, nil
	}
	if err != nil || data == "" {
		return nil, tracerr.Wrap(err)
	}
	session, err := db.GetCTXSession(c)
	if err != nil {
		return nil, err
	}
	ret, err := db.LoadSession(session)
	if err != nil {
		session.Options.MaxAge = -1 // delete invalid
		return nil, db.SaveCTXSession(c)
	}
	return ret, nil
}

func (impl *APIImpl) parseRequest(c *gin.Context) (*sn.Request, error) {
	lang := c.DefaultQuery("lang", "en-US")
	logger := log.New().WithField("ip", c.ClientIP())
	if len(c.Request.URL.Query()) != 0 {
		logger = logger.WithField("query", c.Request.URL.Query())
	}
	logger = logger.WithField("path", c.Request.URL.Path)
	c.SetSameSite(http.SameSiteStrictMode)
	req := &sn.Request{
		Logger:     logger,
		Context:    c,
		Translator: translator.NewLocalizer(lang),
		Perm:       make(map[uuid.UUID]*sn.PermEntry),
	}
	req.Perm[sn.Skynet.ID.Get(sn.PermGuestID)] = &sn.PermEntry{
		ID:        sn.Skynet.ID.Get(sn.PermGuestID),
		Name:      "guest",
		Note:      "all guest user",
		Perm:      sn.PermAll,
		Origin:    nil,
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}

	session, err := impl.checkSignIn(c)
	if err != nil {
		return req, err
	}
	if session != nil {
		req.Perm[sn.Skynet.ID.Get(sn.PermUserID)] = &sn.PermEntry{
			ID:        sn.Skynet.ID.Get(sn.PermUserID),
			Name:      "user",
			Note:      "all login user",
			Perm:      sn.PermAll,
			Origin:    nil,
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		}
		req.ID = session.ID
		if req.ID != uuid.Nil {
			req.Logger = req.Logger.WithField("user", req.ID)
		}
		merged, err := sn.Skynet.Permission.GetUserMerged(req.ID)
		if err != nil {
			return req, err
		}
		for k, v := range merged {
			req.Perm[k] = v
		}
	}
	return req, nil
}

func (impl *APIImpl) handlerFunc(i *sn.APIItem) func(c *gin.Context) {
	handler := func(c *gin.Context) (*sn.Request, *sn.Response, error) {
		req, err := impl.parseRequest(c)
		if err != nil {
			return req, nil, err
		}
		// perm
		ok := false
		if i.Checker != nil {
			ok = i.Checker(req.Perm)
		} else {
			if i.Perm != nil {
				ok = i.Perm.Check(req.Perm)
			}
		}
		if ok {
			rsp, err := i.Func(req)
			return req, rsp, err
		} else {
			return req, sn.ResponseForbidden, nil
		}
	}
	return func(c *gin.Context) {
		req, rsp, err := handler(c)
		if rsp == nil {
			rsp = sn.ResponseError
		}
		if err != nil {
			log.WrapEntry(req.Logger, err).Error("Request handler error")
		}
		rsp = rsp.Parse(req.Translator)
		if rsp.HTTPCode == 403 {
			c.String(403, "Permission denied")
			return
		} else if rsp.HTTPCode != 200 {
			c.AbortWithStatus(rsp.HTTPCode)
			return
		}
		c.JSON(rsp.HTTPCode, rsp)
	}
}

func (impl *APIImpl) Add(r *gin.RouterGroup, item []*sn.APIItem) {
	for _, v := range item {
		if v.Perm == nil && v.Checker == nil {
			log.New().Fatalf("API %v do not have permission", v.Path)
		}
		fun := impl.handlerFunc(v)
		switch v.Method {
		case sn.APIGet:
			r.GET(v.Path, fun)
		case sn.APIPost:
			r.POST(v.Path, fun)
		case sn.APIPut:
			r.PUT(v.Path, fun)
		case sn.APIPatch:
			r.PATCH(v.Path, fun)
		case sn.APIDelete:
			r.DELETE(v.Path, fun)
		case sn.APIOptions:
			r.OPTIONS(v.Path, fun)
		case sn.APIHead:
			r.HEAD(v.Path, fun)
		case sn.APIAny:
			r.Any(v.Path, fun)
		}
	}
	impl.api = append(impl.api, item...)
}

func (impl *APIImpl) Get() []*sn.APIItem {
	return impl.api
}

type MenuImpl struct {
	menu []*sn.MenuItem
}

func (impl *MenuImpl) GetAll() []*sn.MenuItem {
	return impl.menu
}

func (impl *MenuImpl) GetByID(id uuid.UUID) *sn.MenuItem {
	var dfs func([]*sn.MenuItem) *sn.MenuItem
	dfs = func(this []*sn.MenuItem) *sn.MenuItem {
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
	return dfs(impl.menu)
}

func (impl *MenuImpl) Add(item *sn.MenuItem, parent uuid.UUID) bool {
	if parent == uuid.Nil {
		impl.menu = append(impl.menu, item)
		return true
	}

	p := impl.GetByID(parent)
	if p == nil {
		return false
	}
	p.Children = append(p.Children, item)
	return true
}

func Init(r *gin.RouterGroup) {
	api := new(APIImpl)
	api.Add(r, initAPI())
	sn.Skynet.API = api
	menu := new(MenuImpl)
	menu.menu = initMenu()
	sn.Skynet.Menu = menu
}

func success(l *logrus.Entry, s string) {
	l.Info(s)
	sn.Skynet.Notification.New(sn.NotifySuccess, "Skynet log", s, utils.MustMarshal(l.Data))
}

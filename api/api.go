package api

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/handler"
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/translator"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

var (
	rspOK           = &Response{Code: CodeOK}
	rspParamInvalid = &Response{HTTPCode: 400}
)

var (
	API    = []*APIItem{}
	Menu   = []*MenuItem{}
	Router *gin.RouterGroup
	Locker sync.Mutex
)

// APIMethod is http method for API.
type APIMethod int32

const (
	// APIGet represents HTTP Get method
	APIGet APIMethod = iota
	// APIPost represents HTTP Post method
	APIPost
	// APIPut represents HTTP Put method
	APIPut
	// APIPatch represents HTTP Patch method
	APIPatch
	// APIDelete represents HTTP Delete method
	APIDelete
	// APIOptions represents HTTP Options method
	APIOptions
	// APIHead represents HTTP Head method
	APIHead
	// APIAny represents any HTTP method
	APIAny
)

// APIFunc is API function type.
//
// id is current user id when user signin, otherwise uuid.Nil.
//
// return http response and error.
type APIFunc func(r *Request) (*Response, error)

// APIItem is API item struct.
type APIItem struct {
	Path    string          // API path
	Method  APIMethod       // API http method
	Perm    *handler.Perm   // API permission
	Func    APIFunc         // API function
	Checker PermCheckerFunc // custom perm checker, invoke when default perm check failed
}

// SNCheckerFunc is permission checker function type.
type PermCheckerFunc func(perm map[uuid.UUID]*handler.Perm) bool

type MenuItem struct {
	ID        uuid.UUID
	Name      string
	Path      string
	Icon      string
	OmitEmpty bool
	Children  []*MenuItem
	Perm      *handler.Perm
	Checker   PermCheckerFunc
}

func (m *MenuItem) Check(p map[uuid.UUID]*handler.Perm) bool {
	if m.Checker == nil && (m.Perm == nil || m.Perm.ID == uuid.Nil) { // menu group
		return true
	} else {
		ok := false
		if m.Perm != nil {
			ok = CheckPerm(p, m.Perm)
		}
		if m.Checker != nil {
			ok = m.Checker(p)
		}
		return ok
	}
}

type RspCode int32

const (
	CodeOK RspCode = iota
	CodeInvalidUserOrPass
	CodeUsernameExist
	CodeInvalidRecaptcha
	CodeRestarting
	CodeGroupNotExist
	CodeGroupNameExist
	CodeRootNotAllowedUpdate
	CodeRootNotAllowedDelete
	CodePluginNotExist
	CodePluginFormatError
	CodePluginExist

	CodeMax
)

func (c RspCode) String(t *i18n.Localizer) string {
	return translator.TranslateString(t, RspMsg[c])
}

var RspMsg = [CodeMax]string{
	"response.success",
	"response.user.invalid",
	"response.user.exist",
	"response.recaptcha.invalid",
	"response.restart",
	"response.group.notexist",
	"response.group.exist",
	"response.group.rootupdate",
	"response.group.rootdelete",
	"response.plugin.notexist",
	"response.plugin.formaterror",
	"response.plugin.exist",
}

type Response struct {
	HTTPCode int     `json:"-"`
	Code     RspCode `json:"code"`
	Msg      string  `json:"msg"`
	Data     any     `json:"data,omitempty"`
}

func NewPageResponse(data any, total int64) *Response {
	return &Response{
		Code: CodeOK,
		Data: gin.H{
			"data":  data,
			"total": total,
		},
	}
}

type Request struct {
	ID         uuid.UUID
	Context    *gin.Context
	Logger     *logrus.Entry
	Translator *i18n.Localizer
	Perm       map[uuid.UUID]*handler.Perm
}

func (req *Request) ShouldBind(to any) error {
	return tracerr.Wrap(req.Context.ShouldBind(to))
}

func (req *Request) ShouldBindQuery(to any) error {
	return tracerr.Wrap(req.Context.ShouldBindQuery(to))
}

func (req *Request) ShouldBindUri(to any) error {
	return tracerr.Wrap(req.Context.ShouldBindUri(to))
}

func Init(r *gin.RouterGroup) {
	Router = r
	AddAPI(initAPI())
	Locker.Lock()
	defer Locker.Unlock()
	Menu = initMenu()
}

func CheckPerm(perm map[uuid.UUID]*handler.Perm, target *handler.Perm) bool {
	if target.ID == db.GetDefaultID(db.PermGuestID) {
		return true
	}
	if perm != nil {
		if target.ID == db.GetDefaultID(db.PermUserID) {
			return true
		}
		if p, ok := perm[target.ID]; ok { // user permission check first
			return (p.Perm & target.Perm) == target.Perm
		}
		if p, ok := perm[db.GetDefaultID(db.PermAllID)]; ok {
			return (p.Perm & target.Perm) == target.Perm
		}
	}
	return false // fail safe
}

func ParseRequest(c *gin.Context) (*Request, error) {
	lang := c.DefaultQuery("lang", "en-US")
	logger := log.New().WithField("ip", c.ClientIP())
	if len(c.Request.URL.Query()) != 0 {
		logger = logger.WithField("query", c.Request.URL.Query())
	}
	logger = logger.WithField("path", c.Request.URL.Path)
	req := &Request{
		Logger:     logger,
		Context:    c,
		Translator: translator.NewLocalizer(lang),
	}

	session, err := checkSignIn(c)
	if err != nil {
		return req, err
	}
	if session != nil {
		req.ID = session.ID
		if req.ID != uuid.Nil {
			req.Logger = req.Logger.WithField("user", req.ID)
		}
		req.Perm, err = handler.Permission.GetUserMerged(req.ID)
		if err != nil {
			return req, err
		}
	}
	return req, nil
}

func handlerFunc(i *APIItem) func(c *gin.Context) {
	handler := func(c *gin.Context) (*Request, *Response, error) {
		req, err := ParseRequest(c)
		if err != nil {
			return req, nil, err
		}
		if !sn.Running {
			return req, &Response{Code: CodeRestarting}, nil
		}
		ok := CheckPerm(req.Perm, i.Perm)
		if !ok && i.Checker != nil && i.Checker(req.Perm) {
			ok = true
		}
		if ok {
			rsp, err := i.Func(req)
			return req, rsp, err
		} else {
			return req, &Response{HTTPCode: 403}, nil
		}
	}
	return func(c *gin.Context) {
		req, rsp, err := handler(c)
		if rsp == nil {
			rsp = &Response{HTTPCode: 500}
		}
		if rsp.Msg == "" {
			if int(rsp.Code) < len(RspMsg) {
				rsp.Msg = rsp.Code.String(req.Translator)
			}
		}
		if err != nil {
			if rsp.HTTPCode == 0 {
				rsp.HTTPCode = 500
			}
			log.MergeEntry(req.Logger, log.NewEntry(err)).Error("Request handler error")
			c.AbortWithStatus(rsp.HTTPCode)
			return
		}
		if rsp.HTTPCode == 403 {
			c.String(403, "Permission denied")
			return
		} else if rsp.HTTPCode != 0 && rsp.HTTPCode != 200 {
			c.AbortWithStatus(rsp.HTTPCode)
			return
		}
		c.JSON(rsp.HTTPCode, rsp)
	}
}

func checkSignIn(c *gin.Context) (*db.SessionData, error) {
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

func AddAPI(item []*APIItem) {
	Locker.Lock()
	defer Locker.Unlock()
	for _, v := range item {
		if v.Perm == nil || v.Perm.ID == uuid.Nil {
			log.New().Warn("API %v do not have permission", v.Path)
		}
		fun := handlerFunc(v)
		switch v.Method {
		case APIGet:
			Router.GET(v.Path, fun)
		case APIPost:
			Router.POST(v.Path, fun)
		case APIPut:
			Router.PUT(v.Path, fun)
		case APIPatch:
			Router.PATCH(v.Path, fun)
		case APIDelete:
			Router.DELETE(v.Path, fun)
		case APIOptions:
			Router.OPTIONS(v.Path, fun)
		case APIHead:
			Router.HEAD(v.Path, fun)
		case APIAny:
			Router.Any(v.Path, fun)
		}
	}
	API = append(API, item...)
}

func getMenuByID(id uuid.UUID) *MenuItem {
	var dfs func([]*MenuItem) *MenuItem
	dfs = func(this []*MenuItem) *MenuItem {
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
	return dfs(Menu)
}

func AddMenu(item *MenuItem, parent uuid.UUID) bool {
	Locker.Lock()
	defer Locker.Unlock()
	if parent == uuid.Nil {
		Menu = append(Menu, item)
		return true
	}

	p := getMenuByID(parent)
	if p == nil {
		return false
	}
	p.Children = append(p.Children, item)
	return true
}

func success(l *logrus.Entry, s string) {
	l.Info(s)
	handler.Notification.New(db.NotifySuccess, "Skynet log", s, utils.MustMarshal(l.Data))
}

type paginationParam struct {
	Page int `form:"page,default=1" binding:"min=1"`
	Size int `form:"size,default=10" binding:"min=1"`
}

func (p *paginationParam) ToCondition() *db.Condition {
	return &db.Condition{
		Limit:  p.Size,
		Offset: (p.Page - 1) * p.Size,
	}
}

type createdParam struct {
	CreatedSort  string `form:"createdSort" binding:"omitempty,oneof=asc desc"`
	CreatedStart int64  `form:"createdStart" binding:"min=0"`
	CreatedEnd   int64  `form:"createdEnd" binding:"min=0"`
}

func (p *createdParam) ToCondition() *db.Condition {
	ret := new(db.Condition)
	now := time.Now().UnixMilli()
	if p.CreatedEnd == 0 {
		p.CreatedEnd = now
	}
	if !(p.CreatedStart == 0 && p.CreatedEnd == now) {
		ret.And("created_at BETWEEN ? AND ?", p.CreatedStart, p.CreatedEnd)
	}
	if p.CreatedSort != "" {
		ret.Order = []any{"created_at " + p.CreatedSort}
	}
	return ret
}

type updatedParam struct {
	UpdatedSort  string `form:"updatedSort" binding:"omitempty,oneof=asc desc"`
	UpdatedStart int64  `form:"updatedStart" binding:"min=0"`
	UpdatedEnd   int64  `form:"updatedEnd" binding:"min=0"`
}

func (p *updatedParam) ToCondition() *db.Condition {
	ret := new(db.Condition)
	now := time.Now().UnixMilli()
	if p.UpdatedEnd == 0 {
		p.UpdatedEnd = now
	}
	if !(p.UpdatedStart == 0 && p.UpdatedEnd == now) {
		ret.And("updated_at BETWEEN ? AND ?", p.UpdatedStart, p.UpdatedEnd)
	}
	if p.UpdatedSort != "" {
		ret.Order = []any{"updated_at " + p.UpdatedSort}
	}
	return ret
}

type idURI struct {
	ID string `uri:"id" binding:"required,uuid"`
}

package sn

import (
	"regexp"
	"strings"
	"time"

	"github.com/MXWXZ/skynet/translator"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

var (
	ResponseOK           = &Response{}
	ResponseParamInvalid = &Response{HTTPCode: 400}
	ResponseForbidden    = &Response{HTTPCode: 403}
	ResponseError        = &Response{HTTPCode: 500}
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

type API interface {
	Get() []*APIItem
	Add(r *gin.RouterGroup, item []*APIItem)
}

type Request struct {
	ID         uuid.UUID
	Context    *gin.Context
	Logger     *logrus.Entry
	Translator *i18n.Localizer
	Perm       map[uuid.UUID]*PermEntry
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

//go:generate stringer -type=ResponseCode

type ResponseCode int32

const (
	CodeSuccess ResponseCode = iota
	CodeUserInvalid
	CodeUserNotexist
	CodeUserExist
	CodeRecaptchaInvalid
	CodeGroupNotexist
	CodeGroupExist
	CodeGroupRootupdate
	CodeGroupRootdelete
	CodePermissionNotexist
	CodePluginNotExist
	CodePluginLoaded

	CodeMax
)

func (c ResponseCode) GetMsg() string {
	re := regexp.MustCompile("([A-Z])")
	ret, _ := strings.CutPrefix(c.String(), "Code")
	ret = re.ReplaceAllString(ret, ".$1")
	ret = "response" + strings.ToLower(ret)
	return ret
}

type Response struct {
	HTTPCode int          `json:"-"`
	Code     ResponseCode `json:"code"`
	Msg      string       `json:"msg"`
	Data     any          `json:"data,omitempty"`
}

func (rsp *Response) Parse(t *i18n.Localizer) *Response {
	ret := &Response{
		HTTPCode: rsp.HTTPCode,
		Code:     rsp.Code,
		Msg:      rsp.Msg,
		Data:     rsp.Data,
	}
	if ret.Msg == "" {
		ret.Msg = ret.Code.GetMsg()
	}
	ret.Msg = translator.TranslateString(t, ret.Msg)
	if ret.HTTPCode == 0 {
		ret.HTTPCode = 200
	}
	return ret
}

// APIFunc is API function type.
type APIFunc func(r *Request) (*Response, error)

// APIItem is API item struct.
type APIItem struct {
	Path    string          // API path
	Method  APIMethod       // API http method
	Perm    *PermEntry      // API permission
	Func    APIFunc         // API function
	Checker PermCheckerFunc // custom perm checker, ignore default perm check
}

// Param helper

type PaginationParam struct {
	Page int `form:"page,default=1" binding:"min=1"`
	Size int `form:"size,default=10" binding:"min=1"`
}

func (p *PaginationParam) ToCondition() *Condition {
	return &Condition{
		Limit:  p.Size,
		Offset: (p.Page - 1) * p.Size,
	}
}

type CreatedParam struct {
	CreatedSort  string `form:"createdSort" binding:"omitempty,oneof=asc desc"`
	CreatedStart int64  `form:"createdStart" binding:"min=0"`
	CreatedEnd   int64  `form:"createdEnd" binding:"min=0"`
}

func (p *CreatedParam) ToCondition() *Condition {
	ret := new(Condition)
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

type UpdatedParam struct {
	UpdatedSort  string `form:"updatedSort" binding:"omitempty,oneof=asc desc"`
	UpdatedStart int64  `form:"updatedStart" binding:"min=0"`
	UpdatedEnd   int64  `form:"updatedEnd" binding:"min=0"`
}

func (p *UpdatedParam) ToCondition() *Condition {
	ret := new(Condition)
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

type IDURI struct {
	ID string `uri:"id" binding:"required,uuid"`
}

func (i *IDURI) Parse() (uuid.UUID, error) {
	ret, err := uuid.Parse(i.ID)
	return ret, tracerr.Wrap(err)
}

func NewPageResponse(data any, total int64) *Response {
	return &Response{
		Data: gin.H{
			"data":  data,
			"total": total,
		},
	}
}

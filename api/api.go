package api

import (
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type siteAPI struct {
	router *gin.RouterGroup
	api    []*sn.SNAPIItem
}

func (s *siteAPI) GetRouter() *gin.RouterGroup {
	return s.router
}

func (s *siteAPI) GetAPI() []*sn.SNAPIItem {
	return s.api
}

func (s *siteAPI) AddAPI(i []*sn.SNAPIItem) {
	for _, v := range i {
		var fun func(c *gin.Context)
		switch v.Role {
		case sn.RoleEmpty:
			fun = func(f sn.SNAPIFunc) func(c *gin.Context) {
				return func(c *gin.Context) {
					code, err := f(c, nil)
					if err != nil {
						log.Error(err)
						c.AbortWithStatus(code)
						return
					}
				}
			}(v.Func)
		case sn.RoleUser:
			fun = utils.WithSignIn(v.Func, false)
		case sn.RoleAdmin:
			fun = utils.WithAdmin(v.Func, false)
		}
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
	s.api = append(s.api, i...)
}

// NewAPI returns new API object.
func NewAPI(r *gin.RouterGroup) sn.SNAPI {
	var ret siteAPI
	ret.router = r
	ret.AddAPI(api)
	return &ret
}

type paginationParam struct {
	Order string `form:"order,default=asc" binding:"oneof=asc desc"`
	Page  int    `form:"page,default=1" binding:"min=1"`
	Size  int    `form:"size,default=10"`
}

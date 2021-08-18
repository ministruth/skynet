package page

import (
	"html/template"
	"net/http"
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
)

type sitePage struct {
	renderer multitemplate.Renderer
	router   *gin.RouterGroup
	page     []*sn.SNPageItem
	navbar   []*sn.SNNavItem
}

func NewPage(t multitemplate.Renderer, r *gin.RouterGroup) sn.SNPage {
	var ret sitePage
	ret.renderer = t
	ret.router = r
	ret.AddNav(navbar)
	ret.AddPage(pages)
	return &ret
}

func (s *sitePage) GetDefaultFunc() template.FuncMap {
	return defaultFunc
}

func (s *sitePage) GetDefaultPath() *sn.SNPathItem {
	return defaultPath
}

func (s *sitePage) GetRouter() *gin.RouterGroup {
	return s.router
}

func (s *sitePage) GetNav() []*sn.SNNavItem {
	return s.navbar
}

func (s *sitePage) GetPage() []*sn.SNPageItem {
	return s.page
}

func (s *sitePage) AddNav(i []*sn.SNNavItem) {
	s.navbar = append(s.navbar, i...)
	sn.SNNavSort(s.navbar).Sort()
}

func (r *sitePage) RenderSingle(c *gin.Context, u *sn.User, p *sn.SNPageItem) {
	p.Param["_nonce"] = c.Keys["nonce"]
	p.Param["_csrftoken"] = csrf.Token(c.Request)

	ret := true
	if p.AfterRenderPrepare != nil {
		ret = p.AfterRenderPrepare(c, u, p)
	}
	if ret {
		c.HTML(http.StatusOK, p.TplName, p.Param)
	}
}

func (r *sitePage) Render(c *gin.Context, u *sn.User, p *sn.SNPageItem) {
	avatar, err := utils.ConvertWebp(u.Avatar)
	if err != nil {
		log.Fatal(err)
	}
	tmpNav := make([]*sn.SNNavItem, len(r.navbar))
	copier.CopyWithOption(&tmpNav, &r.navbar, copier.Option{DeepCopy: true})

	var activateLink func(i []*sn.SNNavItem) bool
	activateLink = func(i []*sn.SNNavItem) bool {
		for _, v := range i {
			if v.Link == p.Link {
				v.Active = true
				return true
			}
			if activateLink(v.Child) {
				v.Active = true
				v.Open = true
				return true
			}
		}
		return false
	}
	activateLink(tmpNav)
	var navPrepare func(i []*sn.SNNavItem)
	navPrepare = func(i []*sn.SNNavItem) {
		for _, v := range i {
			if v.RenderPrepare != nil {
				if !v.RenderPrepare(c, v, i) {
					return
				}
			}
			navPrepare(v.Child)
		}
	}
	navPrepare(tmpNav)

	p.Param["_title"] = p.Title
	p.Param["_name"] = p.Name
	p.Param["_id"] = u.ID
	p.Param["_username"] = u.Username
	p.Param["_avatar"] = avatar.Base64()
	p.Param["_navbar"] = tmpNav
	p.Param["_path"] = p.Path
	p.Param["_role"] = u.Role
	p.Param["_version"] = sn.VERSION
	for _, v := range p.QueryParam {
		p.Param["_"+v] = c.Query(v)
	}

	r.RenderSingle(c, u, p)
}

func (s *sitePage) AddPage(i []*sn.SNPageItem) {
	for _, v := range i {
		s.renderer.AddFromFilesFuncs(v.TplName, v.FuncMap, v.Files...)
		renderer := func(v *sn.SNPageItem) func(c *gin.Context, u *sn.User) (int, error) {
			return func(c *gin.Context, u *sn.User) (int, error) {
				ret := true
				if v.Param == nil {
					v.Param = make(map[string]interface{})
				}
				if v.BeforeRender != nil {
					ret = v.BeforeRender(c, u, v)
				}
				if ret {
					s.Render(c, u, v)
				}
				if v.AfterRender != nil {
					v.AfterRender(c, u, v)
				}
				return 0, nil
			}
		}
		switch v.Role {
		case sn.RoleEmpty:
			s.router.GET(v.Link, func(v *sn.SNPageItem) func(c *gin.Context) {
				return func(c *gin.Context) {
					if v.Param == nil {
						v.Param = make(map[string]interface{})
					}
					ret := true
					if v.BeforeRender != nil {
						ret = v.BeforeRender(c, nil, v)
					}
					if ret {
						s.RenderSingle(c, nil, v)
					}
					if v.AfterRender != nil {
						v.AfterRender(c, nil, v)
					}
				}
			}(v))
		case sn.RoleUser:
			s.router.GET(v.Link, utils.WithSignIn(renderer(v), true))
		case sn.RoleAdmin:
			s.router.GET(v.Link, utils.WithAdmin(renderer(v), true))
		}
	}
	s.page = append(s.page, i...)
}

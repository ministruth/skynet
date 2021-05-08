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
	ret.AddNavItem(navbar)
	ret.AddPageItem(pages)
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

func (s *sitePage) GetNavItem() []*sn.SNNavItem {
	return s.navbar
}

func (s *sitePage) GetPageItem() []*sn.SNPageItem {
	return s.page
}

func (s *sitePage) AddNavItem(i []*sn.SNNavItem) {
	s.navbar = append(s.navbar, i...)
	sn.SNNavSort(s.navbar).Sort()
}

func (r *sitePage) RenderSingle(c *gin.Context, page string, p gin.H) {
	if p == nil {
		p = make(map[string]interface{})
	}
	p["_nonce"] = c.Keys["nonce"]
	p["_csrftoken"] = csrf.Token(c.Request)

	c.HTML(http.StatusOK, page, p)
}

func (r *sitePage) Render(c *gin.Context, u *sn.Users, p *sn.SNPageItem) {
	avatar, err := utils.PicFromByte(u.Avatar)
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

	if p.Param == nil {
		p.Param = make(map[string]interface{})
	}
	p.Param["_title"] = p.Title
	p.Param["_name"] = p.Name
	p.Param["_id"] = u.ID
	p.Param["_username"] = u.Username
	p.Param["_avatar"] = avatar.Base64()
	p.Param["_navbar"] = tmpNav
	p.Param["_path"] = p.Path
	p.Param["_role"] = u.Role
	p.Param["_version"] = sn.VERSION

	r.RenderSingle(c, p.TplName, p.Param)
}

func (s *sitePage) AddPageItem(i []*sn.SNPageItem) {
	for _, v := range i {
		s.renderer.AddFromFilesFuncs(v.TplName, v.FuncMap, v.Files...)
		renderer := func(v *sn.SNPageItem) func(c *gin.Context, u *sn.Users) {
			return func(c *gin.Context, u *sn.Users) {
				ret := true
				if v.BeforeRender != nil {
					ret = v.BeforeRender(c, u, v)
				}
				if ret {
					s.Render(c, u, v)
				}
				if v.AfterRender != nil {
					v.AfterRender(c, u, v)
				}
			}
		}
		switch v.Role {
		case sn.RoleEmpty:
			s.router.GET(v.Link, func(v *sn.SNPageItem) func(c *gin.Context) {
				return func(c *gin.Context) {
					ret := true
					if v.BeforeRender != nil {
						ret = v.BeforeRender(c, nil, v)
					}
					if ret {
						s.RenderSingle(c, v.TplName, v.Param)
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

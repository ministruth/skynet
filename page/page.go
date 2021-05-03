package page

import (
	"html/template"
	"log"
	"net/http"
	"skynet/api"
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/jinzhu/copier"
)

func NewPage(t multitemplate.Renderer) sn.SNPage {
	var ret sitePage
	ret.renderer = t
	ret.page = []*sn.SNPageItem{
		// TODO: Add new page
		{
			Name: "Dashboard",
			Link: "/dashboard",
			Icon: "fa-tachometer-alt",
			Role: 1,
		},
		{
			Name: "Service",
			Link: "#",
			Icon: "fa-briefcase",
			Role: 1,
		},
		{
			Name: "Plugin",
			Link: "#",
			Icon: "fa-plug",
			Role: 1,
			Child: []*sn.SNPageItem{
				{
					Name: "Manager",
					Link: "/plugin",
					Role: 2,
				},
			},
		},
		{
			Name: "User",
			Link: "/user",
			Icon: "fa-user",
			Role: 2,
		},
		{
			Name: "Setting",
			Link: "/setting",
			Icon: "fa-cog",
			Role: 1,
		},
	}

	// TODO: Add new page
	ret.ParseTemplateStatic("index")
	ret.ParseTemplateStatic("header")
	ret.ParseTemplateStatic("footer")
	ret.ParseTemplate("dashboard")
	ret.ParseTemplate("setting")
	ret.ParseTemplate("user")
	ret.ParseTemplate("deny")
	ret.ParseTemplate("plugin")

	return &ret
}

var defaultFunc = template.FuncMap{
	// TODO: Add page functions
	"api":  apiVersion,
	"time": formatTime,
}

func PageRouter(r *gin.RouterGroup) {
	// TODO: Add new page
	r.GET("/", PageIndex)
	r.GET("/dashboard", utils.NeedSignIn(PageDashboard, true))
	r.GET("/setting", utils.NeedSignIn(PageSetting, true))
	r.GET("/user", utils.NeedAdmin(PageUser, true))
	r.GET("/deny", utils.NeedSignIn(PageDeny, true))
	r.GET("/plugin", utils.NeedAdmin(PagePlugin, true))
}

type sitePage struct {
	renderer multitemplate.Renderer
	page     []*sn.SNPageItem
}

func (r *sitePage) GetPage() []*sn.SNPageItem {
	return r.page
}

func (r *sitePage) AddTemplate(name string, files ...string) {
	r.renderer.AddFromFilesFuncs(name, defaultFunc, files...)
}

func (r *sitePage) RenderSingle(c *gin.Context, page string, p gin.H) {
	p["_nonce"] = c.Keys["nonce"]
	p["_csrftoken"] = csrf.Token(c.Request)

	c.HTML(http.StatusOK, page, p)
}

func (r *sitePage) Render(c *gin.Context, page string, title string, name string, link string, u *sn.Users, p gin.H) {
	avatar, err := utils.PicFromByte(u.Avatar)
	if err != nil {
		log.Fatal(err)
	}
	snpath := sn.Skynet.Page.GetPage()
	tmpPath := make([]*sn.SNPageItem, len(snpath))
	copier.CopyWithOption(&tmpPath, &snpath, copier.Option{DeepCopy: true})
	for i := range tmpPath {
		if tmpPath[i].Link == link {
			tmpPath[i].Active = true
		}
		for j := range tmpPath[i].Child {
			if tmpPath[i].Child[j].Link == link {
				tmpPath[i].Child[j].Active = true
				tmpPath[i].Active = true
				tmpPath[i].Open = true
			}
		}
	}

	p["_title"] = title
	p["_name"] = name
	p["_id"] = u.ID
	p["_username"] = u.Username
	p["_avatar"] = avatar.Base64()
	p["_sitepath"] = tmpPath
	p["_role"] = u.Role
	p["_version"] = utils.VERSION

	r.RenderSingle(c, page, p)
}

func (r *sitePage) ParseTemplateStatic(n string) {
	r.renderer.AddFromFilesFuncs(n, defaultFunc, "templates/"+n+".tmpl", "templates/header.tmpl", "templates/footer.tmpl")
}

func (r *sitePage) ParseTemplate(n string) {
	r.renderer.AddFromFilesFuncs(n, defaultFunc, "templates/home.tmpl", "templates/"+n+".tmpl", "templates/header.tmpl", "templates/footer.tmpl")
}

func apiVersion(in string) (string, error) {
	return api.APIVERSION + in, nil
}

func formatTime(t time.Time) (string, error) {
	return t.Format("2006-01-02 15:04:05"), nil
}

func PageDeny(c *gin.Context, u *sn.Users) {
	sn.Skynet.Page.Render(c, "deny", "Skynet | Permission Denied", "Permission Denied", "", u, gin.H{
		"_path": append(sn.SNDefaultPath, &sn.SNPageItem{
			Name:   "Permission Denied",
			Active: true,
		}),
	})
}

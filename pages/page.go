package pages

import (
	"fmt"
	"log"
	"net/http"
	"skynet/db"
	"skynet/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
)

type pagePath struct {
	Name   string
	Active bool
	Link   string
	Icon   string
}

var defaultPath = []pagePath{
	{
		Name: "Home",
		Link: "/",
	},
}

var sitePath = []pagePath{
	{
		Name: "Dashboard",
		Link: "/dashboard",
		Icon: "fa-tachometer-alt",
	},
	{
		Name: "Setting",
		Link: "/setting",
		Icon: "fa-cog",
	},
}

func PageRouter(r *gin.RouterGroup) {
	r.GET("/", PageIndex)
	r.GET("/dashboard", utils.NeedSignIn(PageDashboard, true))
	r.GET("/setting", utils.NeedSignIn(PageSetting, true))
}

func render(c *gin.Context, page string, param gin.H) {
	tmp := param
	tmp["nonce"] = c.Keys["nonce"]
	tmp["csrftoken"] = csrf.Token(c.Request)
	c.HTML(http.StatusOK, page, tmp)
}

func commonParam(title string, name string, u *db.Users, p gin.H) gin.H {
	ret := p
	avatar, err := utils.ConvertPictureBase64(u.Avatar)
	if err != nil {
		log.Fatal(err)
	}
	tmpPath := make([]pagePath, len(sitePath))
	copy(tmpPath, sitePath)
	for i := range tmpPath {
		if tmpPath[i].Name == name {
			tmpPath[i].Active = true
		}
	}
	fmt.Println(tmpPath)

	ret["title"] = title
	ret["name"] = name
	ret["username"] = u.Username
	ret["avatar"] = avatar
	ret["sitepath"] = tmpPath
	return ret
}

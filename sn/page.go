package sn

import "github.com/gin-gonic/gin"

type SNPage interface {
	GetPage() []*SNPageItem
	AddTemplate(name string, files ...string)
	Render(c *gin.Context, page string, title string, name string, link string, u *Users, p gin.H)
	RenderSingle(c *gin.Context, page string, p gin.H)
}

type SNPageItem struct {
	Name   string
	Active bool
	Open   bool
	Link   string
	Icon   string
	Role   UserRole
	Child  []*SNPageItem
}

var SNDefaultPath = []*SNPageItem{
	{
		Name: "Home",
		Link: "/",
	},
}

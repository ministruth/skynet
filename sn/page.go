package sn

import (
	"html/template"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
)

type SNPage interface {
	GetRouter() *gin.RouterGroup
	GetNavItem() []*SNNavItem
	GetPageItem() []*SNPageItem
	AddNavItem(i []*SNNavItem)
	AddPageItem(i []*SNPageItem)
	GetDefaultFunc() template.FuncMap
	GetDefaultPath() *SNPathItem
}

type SNPathItem struct {
	Name   string
	Link   string
	Active bool
	Child  *SNPathItem
}

func (i SNPathItem) WithChild(c []*SNPathItem) *SNPathItem {
	cur := &i
	for _, v := range c {
		cur.Child = v
		cur = cur.Child
	}
	return &i
}

type SNNavItem struct {
	Priority      int
	Name          string
	Active        bool
	Open          bool
	Link          string
	Icon          string
	Role          UserRole
	Badge         string
	BadgeClass    string
	RenderPrepare func(*gin.Context, *SNNavItem, []*SNNavItem) bool
	Child         []*SNNavItem
}

func (i *SNNavItem) SortChild() {
	for _, v := range i.Child {
		v.SortChild()
	}
	sort.Stable(SNNavSort(i.Child))
}

type SNNavSort []*SNNavItem

func (a SNNavSort) Sort() {
	for _, v := range a {
		v.SortChild()
	}
	sort.Stable(a)
}
func (s SNNavSort) Len() int           { return len(s) }
func (s SNNavSort) Less(i, j int) bool { return s[i].Priority < s[j].Priority }
func (s SNNavSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type SNRenderHookFunc func(*gin.Context, *User, *SNPageItem) bool
type SNPageItem struct {
	TplName            string
	Files              []string
	FuncMap            template.FuncMap
	Title              string
	Name               string
	Link               string
	Path               *SNPathItem
	Role               UserRole
	BeforeRender       SNRenderHookFunc
	AfterRenderPrepare SNRenderHookFunc
	AfterRender        SNRenderHookFunc
	QueryParam         []string
	Param              gin.H
}

func (i *SNPageItem) Render(c *gin.Context) {
	c.HTML(http.StatusOK, i.TplName, i.Param)
}

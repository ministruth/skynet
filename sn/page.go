package sn

import (
	"html/template"
	"sort"

	"github.com/gin-gonic/gin"
)

type SNPage interface {
	GetRouter() *gin.RouterGroup
	GetNav() []*SNNavItem
	GetPage() []*SNPageItem
	AddNav(i []*SNNavItem)
	AddPage(i []*SNPageItem)
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

// SNNavItem is struct for navbar item.
type SNNavItem struct {
	ID            string                                            // unique id
	Name          string                                            // show name
	Priority      uint16                                            // inner priority, auto filled by setting
	Active        bool                                              // is activate by default
	Open          bool                                              // is open subnav by default
	Link          string                                            // nav link
	Icon          string                                            // nav icon, use font-awesome icon string
	Role          UserRole                                          // user permission
	Badge         string                                            // nav badge text
	BadgeClass    string                                            // nav badge class, see adminlte3 badge class
	RenderPrepare func(*gin.Context, *SNNavItem, []*SNNavItem) bool // hook when prepare render
	Child         []*SNNavItem                                      // child navbar
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
	Param              gin.H // Page param, will be parse in advance, do not use dynamic value
}

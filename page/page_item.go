package page

import (
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var defaultPath = &sn.SNPathItem{
	Name: "Home",
	Link: "/",
}

func withLayerFiles(n string) []string {
	return []string{"templates/home.tmpl", "templates/" + n + ".tmpl", "templates/header.tmpl", "templates/footer.tmpl"}
}

func withSingleFiles(n string) []string {
	return []string{"templates/" + n + ".tmpl", "templates/header.tmpl", "templates/footer.tmpl"}
}

var pages = []*sn.SNPageItem{
	{
		TplName: "dashboard",
		Files:   withLayerFiles("dashboard"),
		FuncMap: defaultFunc,
		Title:   "Skynet | Dashboard",
		Name:    "Dashboard",
		Link:    "/dashboard",
		Role:    sn.RoleUser,
		Path: defaultPath.WithChild([]*sn.SNPathItem{
			{
				Name:   "Dashboard",
				Active: true,
			},
		}),
	},
	{
		TplName: "index",
		Files:   withSingleFiles("index"),
		FuncMap: defaultFunc,
		Title:   "Skynet",
		Link:    "/",
		Role:    sn.RoleEmpty,
		BeforeRender: func(c *gin.Context, u *sn.Users, v *sn.SNPageItem) bool {
			if data, err := c.Cookie(viper.GetString("session.cookie")); err == nil && data != "" {
				c.Redirect(302, "/dashboard")
				return false
			}
			return true
		},
	},
	{
		TplName: "plugin",
		Files:   withLayerFiles("plugin"),
		FuncMap: defaultFunc,
		Title:   "Skynet | Plugin Manager",
		Name:    "Plugin Manager",
		Link:    "/plugin",
		Role:    sn.RoleAdmin,
		Path: defaultPath.WithChild([]*sn.SNPathItem{
			{
				Name: "Plugin",
				Link: "/plugin",
			},
			{
				Name:   "Manager",
				Active: true,
			},
		}),
		BeforeRender: func(c *gin.Context, u *sn.Users, v *sn.SNPageItem) bool {
			v.Param["plugins"] = sn.Skynet.Plugin.GetAllPlugin()
			return true
		},
	},
	{
		TplName: "setting",
		Files:   withLayerFiles("setting"),
		FuncMap: defaultFunc,
		Title:   "Skynet | Setting",
		Name:    "Setting",
		Link:    "/setting",
		Role:    sn.RoleUser,
		Path: defaultPath.WithChild([]*sn.SNPathItem{
			{
				Name:   "Setting",
				Active: true,
			},
		}),
	},
	{
		TplName: "deny",
		Files:   withLayerFiles("deny"),
		FuncMap: defaultFunc,
		Title:   "Skynet | Permission Denied",
		Name:    "Permission Denied",
		Link:    "/deny",
		Role:    sn.RoleUser,
		Path: defaultPath.WithChild([]*sn.SNPathItem{
			{
				Name:   "Permission Denied",
				Active: true,
			},
		}),
	},
	{
		TplName: "user",
		Files:   withLayerFiles("user"),
		FuncMap: defaultFunc,
		Title:   "Skynet | User",
		Name:    "User",
		Link:    "/user",
		Role:    sn.RoleAdmin,
		Path: defaultPath.WithChild([]*sn.SNPathItem{
			{
				Name:   "User",
				Active: true,
			},
		}),
		BeforeRender: func(c *gin.Context, u *sn.Users, v *sn.SNPageItem) bool {
			users, err := sn.Skynet.User.GetUser()
			if err != nil {
				log.Error(err)
				c.AbortWithStatus(500)
				return false
			}

			type userParam struct {
				sn.Users
				Online bool
			}
			param := make([]userParam, len(users))
			for i := range users {
				s, err := utils.FindSessionsByID(int(users[i].ID))
				if err != nil {
					log.Error(err)
					c.AbortWithStatus(500)
					return false
				}
				param[i].Users = users[i]
				param[i].Online = len(s) != 0
			}
			v.Param["users"] = param
			return true
		},
	},
}

var navbar = []*sn.SNNavItem{
	{
		Priority: 0,
		Name:     "Dashboard",
		Link:     "/dashboard",
		Icon:     "fa-tachometer-alt",
		Role:     sn.RoleUser,
	},
	{
		Priority: 1,
		Name:     "Service",
		Link:     "#",
		Icon:     "fa-briefcase",
		Role:     sn.RoleUser,
	},
	{
		Priority: 2,
		Name:     "Plugin",
		Link:     "#",
		Icon:     "fa-plug",
		Role:     sn.RoleUser,
		Child: []*sn.SNNavItem{
			{
				Priority: 0,
				Name:     "Manager",
				Link:     "/plugin",
				Role:     sn.RoleAdmin,
			},
		},
	},
	{
		Priority: 3,
		Name:     "User",
		Link:     "/user",
		Icon:     "fa-user",
		Role:     sn.RoleAdmin,
	},
	{
		Priority: 4,
		Name:     "Setting",
		Link:     "/setting",
		Icon:     "fa-cog",
		Role:     sn.RoleUser,
	},
}

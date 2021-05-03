package page

import (
	"skynet/sn"

	"github.com/gin-gonic/gin"
)

func PagePlugin(c *gin.Context, u *sn.Users) {
	sn.Skynet.Page.Render(c, "plugin", "Skynet | Plugin Manager", "Plugin Manager", "/plugin", u, gin.H{
		"plugins": sn.Skynet.Plugin.GetAllPlugin(),
		"_path": append(sn.SNDefaultPath, []*sn.SNPageItem{
			{
				Name: "Plugin",
				Link: "/plugin",
			},
			{
				Name:   "Manager",
				Active: true,
			},
		}...),
	})
}

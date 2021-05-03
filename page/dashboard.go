package page

import (
	"skynet/sn"

	"github.com/gin-gonic/gin"
)

func PageDashboard(c *gin.Context, u *sn.Users) {
	sn.Skynet.Page.Render(c, "dashboard", "Skynet | Dashboard", "Dashboard", "/dashboard", u, gin.H{
		"_path": append(sn.SNDefaultPath, &sn.SNPageItem{
			Name:   "Dashboard",
			Active: true,
		}),
	})
}

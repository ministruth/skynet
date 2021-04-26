package pages

import (
	"skynet/utils"

	"github.com/gin-gonic/gin"
)

func PageDashboard(c *gin.Context) {
	u, err := utils.GetUserFromReq(c)
	if err != nil {
		return
	}
	render(c, "dashboard.tmpl", commonParam("Skynet | Dashboard", "Dashboard", u, gin.H{
		"path": append(defaultPath, pagePath{
			Name:   "Dashboard",
			Active: true,
		}),
	}))
}

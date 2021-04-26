package pages

import (
	"skynet/utils"

	"github.com/gin-gonic/gin"
)

func PageSetting(c *gin.Context) {
	u, err := utils.GetUserFromReq(c)
	if err != nil {
		return
	}
	render(c, "setting.tmpl", commonParam("Skynet | Setting", "Setting", u, gin.H{
		"path": append(defaultPath, pagePath{
			Name:   "Setting",
			Active: true,
		}),
	}))
}

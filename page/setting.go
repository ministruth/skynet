package page

import (
	"skynet/sn"

	"github.com/gin-gonic/gin"
)

func PageSetting(c *gin.Context, u *sn.Users) {
	sn.Skynet.Page.Render(c, "setting", "Skynet | Setting", "Setting", "/setting", u, gin.H{
		"_path": append(sn.SNDefaultPath, &sn.SNPageItem{
			Name:   "Setting",
			Active: true,
		}),
	})
}

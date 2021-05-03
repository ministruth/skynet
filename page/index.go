package page

import (
	"skynet/sn"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func PageIndex(c *gin.Context) {
	if data, err := c.Cookie(viper.GetString("session.cookie")); err == nil && data != "" {
		c.Redirect(302, "/dashboard")
	} else {
		sn.Skynet.Page.RenderSingle(c, "index", gin.H{
			"_title": "Skynet",
		})
	}
}

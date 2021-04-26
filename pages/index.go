package pages

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func PageIndex(c *gin.Context) {
	if data, err := c.Cookie(viper.GetString("session.cookie")); err == nil && data != "" {
		c.Redirect(302, "/dashboard")
	} else {
		render(c, "index.tmpl", gin.H{
			"title": "Skynet",
		})
	}
}

package security

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/unrolled/secure"
)

func SecureMiddleware(debug bool) gin.HandlerFunc {
	hosts := strings.Split(viper.GetString("listen.allowhosts"), ",")
	if len(hosts) == 1 && hosts[0] == "" {
		hosts = []string{}
	}

	secureMiddleware := secure.New(secure.Options{
		BrowserXssFilter:        true,
		ContentTypeNosniff:      true,
		FrameDeny:               true,
		IsDevelopment:           debug,
		SSLRedirect:             viper.GetBool("listen.ssl"),
		ContentSecurityPolicy:   viper.GetString("header.csp"),
		ReferrerPolicy:          "same-origin",
		CrossOriginOpenerPolicy: "same-origin",
		AllowedHosts:            hosts,
		AllowedHostsAreRegex:    true,
		SSLProxyHeaders:         map[string]string{"X-Forwarded-Proto": "https"},
		STSSeconds:              31536000,
	})
	return func(c *gin.Context) {
		err := secureMiddleware.Process(c.Writer, c.Request)
		if err != nil {
			c.Abort()
			return
		}

		if status := c.Writer.Status(); status > 300 && status < 399 {
			c.Abort()
		}
	}
}

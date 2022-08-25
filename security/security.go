package security

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/unrolled/secure"
)

func SecureMiddleware() gin.HandlerFunc {
	hosts := strings.Split(viper.GetString("listen.allowhosts"), ",")
	if len(hosts) == 1 && hosts[0] == "" {
		hosts = []string{}
	}
	secureMiddleware := secure.New(secure.Options{
		AllowedHosts:          hosts,
		AllowedHostsAreRegex:  true,
		HostsProxyHeaders:     []string{"X-Forwarded-Hosts"},
		SSLRedirect:           viper.GetBool("listen.ssl"),
		SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
		STSSeconds:            31536000,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: "default-src 'none'; script-src 'unsafe-eval' 'unsafe-inline' 'self'; connect-src 'self'; frame-src www.recaptcha.net/recaptcha/ www.google.com/recaptcha/; img-src 'self' data:; style-src 'self' 'unsafe-inline'; base-uri 'self'; form-action 'self'; font-src 'self'",
		ReferrerPolicy:        "same-origin",
		IsDevelopment:         false,
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

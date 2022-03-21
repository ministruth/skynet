package cmd

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"skynet/api"
	"skynet/handler"
	"skynet/sn"
	"skynet/sn/impl"
	"skynet/sn/utils"
	"strings"
	"syscall"
	"time"

	logrus_stack "github.com/Gurpartap/logrus-stack"
	"github.com/fvbock/endless"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/unrolled/secure"
	"github.com/ztrue/tracerr"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run skynet server",
	Run:   run,
}

//go:embed i18n/*.yml
var i18nFiles embed.FS

func init() {
	rootCmd.AddCommand(runCmd)
}

func run(cmd *cobra.Command, args []string) {
	// logrus hook
	var err error
	var logFile *os.File
	if viper.GetString("log_file") != "" {
		log.SetFormatter(new(log.JSONFormatter))
		logFile, err = os.OpenFile(viper.GetString("log_file"), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0755)
		if err != nil {
			panic(err)
		}
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
		log.AddHook(logrus_stack.StandardHook())
	}
	defer logFile.Close()
	defer log.Info("========== Skynet server end ==========")

	log.Info("========== Skynet server start ==========")
	log.Infof("config file: %s", conf)

	// database
	impl.ConnectRedis()
	log.WithField("addr", viper.GetString("redis.address")).Info("Redis connected")

	impl.ConnectSession()
	log.WithField("prefix", viper.GetString("session.prefix")).Info("Redis session connected")

	impl.ConnectDB()
	log.WithField("path", viper.GetString("database.path")).Info("Database connected")
	log.AddHook(handler.NotificationHook{})

	// skynet handler
	sn.Skynet.Notification = handler.NewNotification()
	sn.Skynet.Permission = handler.NewPermission()
	sn.Skynet.User = handler.NewUser()
	sn.Skynet.Group = handler.NewGroup()
	// setting
	sn.Skynet.Setting, err = handler.NewSetting()
	if err != nil {
		utils.WithTrace(err).Fatal(err)
	}

	// check default settings
	for _, v := range sn.DefaultSetting {
		if v.WarnDefault && viper.Get(v.Name) == v.Value {
			log.Warnf("Setting %v has default value, please modify your config file for safety", v.Name)
		}
		if v.Checker != nil {
			v.Checker(viper.Get(v.Name))
		}
	}

	// check recaptcha
	if viper.GetBool("recaptcha.enable") {
		err = utils.NewReCAPTCHA(viper.GetString("recaptcha.secret"))
		if err != nil {
			utils.WithTrace(err).Fatal(err)
		}
	}

	// i18n
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)
	err = fs.WalkDir(i18nFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return tracerr.Wrap(err)
		}
		if !d.IsDir() {
			b, err := i18nFiles.ReadFile("i18n/" + d.Name())
			if err != nil {
				return tracerr.Wrap(err)
			}
			bundle.MustParseMessageFileBytes(b, d.Name())
			log.Debugf("Language %v loaded", d.Name())
		}
		return nil
	})
	if err != nil {
		utils.WithTrace(err).Fatal(err)
	}
	sn.Skynet.Translator = bundle

	// init gin
	r := gin.Default()
	sn.Skynet.Engine = r

	// security
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
	secureFunc := func() gin.HandlerFunc {
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
	r.Use(secureFunc())

	// csrf
	csrfFunc := func() gin.HandlerFunc {
		return func(c *gin.Context) {
			if c.Request.Method != "GET" {
				token := c.GetHeader("X-CSRF-Token")
				if token == "" {
					c.AbortWithStatus(400)
					return
				}
				ok, err := impl.CheckCSRFToken(token)
				if err != nil {
					utils.WithTrace(err).Error(err)
					c.AbortWithStatus(500)
					return
				}
				if !ok {
					c.AbortWithStatus(400)
					return
				}
			}
			c.Next()
		}
	}
	r.Use(csrfFunc())

	r.ForwardedByClientIP = false // disable ip forward to prevent sproof
	// BUG: gin
	// if !viper.GetBool("proxy.enable") {
	// 	r.ForwardedByClientIP = false // disable ip forward to prevent sproof
	// } else {
	// 	r.ForwardedByClientIP = true
	// 	r.RemoteIPHeaders = []string{viper.GetString("proxy.header")}
	// }

	// session
	sn.Skynet.GetSession().Options(sessions.Options{
		Path:     "/",
		MaxAge:   viper.GetInt("session.expire"),
		Secure:   viper.GetBool("listen.ssl"),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	if !viper.GetBool("debug") {
		err = impl.DeleteSessions(nil)
		if err != nil {
			utils.WithTrace(err).Fatal(err)
		}
	}

	// plugin auth
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		sp := strings.Split(path, "/")
		if strings.HasPrefix(path, "/_plugin/") {
			if len(sp) >= 3 {
				ids, err := api.GetMenuPluginID(c)
				if err != nil {
					utils.WithTrace(err).Error(err)
					c.AbortWithStatus(500)
					return
				}
				for _, v := range ids {
					if v.String() == sp[2] {
						c.Next()
						return
					}
				}
			}
		} else {
			c.Next()
			return
		}
		// prevent plugin guess
		c.String(403, "Permission denied")
		c.Abort()
	})

	// gin middleware is in order, so plugin middleware should use callback instead of direct use
	r.Use(func(c *gin.Context) {
		if errs := sn.Skynet.Plugin.Call(sn.BeforeMiddleware, c); errs != nil {
			for _, v := range errs {
				utils.WithTrace(v).Error(v)
			}
			c.AbortWithStatus(500)
			return
		}
		c.Next()
		sn.Skynet.Plugin.Call(sn.AfterMiddleware, c)
	})

	// 404
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		sp := strings.Split(path, "/")
		if strings.HasPrefix(path, "/api") {
			// prevent route guess
			c.String(403, "Permission denied")
		} else if !strings.Contains(sp[len(sp)-1], ".") {
			// rewrite for dynamic route
			c.Request.URL.Path = "/"
			r.HandleContext(c)
		} else {
			// static file 404
			c.AbortWithStatus(404)
		}
	})

	// static file
	r.Use(static.Serve("/", static.LocalFile("./assets/", true)))

	// api router
	v1 := r.Group("/api")
	sn.Skynet.API = api.NewAPI(v1)

	sn.Skynet.Plugin, err = handler.NewPlugin("plugin")
	if err != nil {
		log.Fatal("Plugin init error: ", err)
	}
	defer sn.Skynet.Plugin.Fini()

	endless.DefaultHammerTime = 1 * time.Second
	server := endless.NewServer(viper.GetString("listen.address"), r)
	server.BeforeBegin = func(add string) {
		sn.Skynet.Running = true
		sn.Skynet.StartTime = time.Now()
		log.Info("Running pid ", syscall.Getpid())
	}
	if !viper.GetBool("listen.ssl") {
		server.ListenAndServe()
	} else {
		server.ListenAndServeTLS(viper.GetString("listen.ssl_cert"), viper.GetString("listen.ssl_key"))
	}
}

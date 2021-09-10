package cmd

import (
	"context"
	"io"
	"net/http"
	"os"
	"skynet/api"
	"skynet/db"
	"skynet/handler"
	"skynet/page"
	"skynet/sn"
	"skynet/sn/utils"
	"strings"
	"syscall"
	"time"

	logrus_stack "github.com/Gurpartap/logrus-stack"
	"github.com/fvbock/endless"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	adapter "github.com/gwatts/gin-adapter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/unrolled/secure"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run skynet server",
	Run:   run,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func connectDB() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(viper.GetInt("database.timeout")))
	defer cancel()

	switch viper.GetString("database.type") {
	case "sqlite":
		sn.Skynet.DB = db.NewDB(ctx, &db.DBConfig{
			Type: db.DBType_Sqlite,
			Path: viper.GetString("database.path"),
		})
	default:
		log.Fatalf("Database type %s not supported", viper.GetString("database.type"))
	}
}

func connectRedis() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(viper.GetInt("redis.timeout")))
	defer cancel()

	sn.Skynet.Redis = db.NewRedis(ctx, &db.RedisConfig{
		Address:  viper.GetString("redis.address"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})
}

func connectSession() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(viper.GetInt("session.timeout")))
	defer cancel()

	sn.Skynet.Session = db.NewSession(ctx, &db.SessionConfig{
		RedisClient: utils.GetRedis(),
		Prefix:      viper.GetString("session.prefix"),
	})
}

func run(cmd *cobra.Command, args []string) {
	// logrus hook
	var err error
	var logFile *os.File
	if viper.GetString("log_file") != "" {
		log.SetFormatter(&log.JSONFormatter{})
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
	connectRedis()
	log.WithFields(log.Fields{
		"addr": viper.GetString("redis.address"),
	}).Info("Redis connected")
	connectSession()
	log.WithFields(log.Fields{
		"prefix": viper.GetString("session.prefix"),
	}).Info("Redis session connected")
	connectDB()
	log.WithFields(log.Fields{
		"path": viper.GetString("database.path"),
	}).Info("Database connected")
	log.AddHook(handler.NotificationHook{})

	// check default settings
	for k, v := range defaultSettings {
		if k[0] == '*' {
			if viper.Get(k[1:]) == v {
				log.Warnf("Setting %v has default value, please modify your config file for safety", k[1:])
			}
		}
	}

	if !viper.GetBool("debug") {
		gin.SetMode(gin.ReleaseMode)
	} else {
		log.Warn("Debug mode is on, make it off when put into production")
	}
	if !viper.GetBool("recaptcha.enable") {
		log.Warn("reCAPTCHA is disabled, enable it when put into production")
	} else {
		err = utils.NewReCAPTCHA(viper.GetString("recaptcha.secret"))
		if err != nil {
			panic(err)
		}
	}
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
		ContentSecurityPolicy: "default-src 'none'; script-src $NONCE; connect-src 'self'; frame-src www.recaptcha.net/recaptcha/ www.google.com/recaptcha/; img-src 'self' data:; style-src 'self'; base-uri 'self'; form-action 'self'; font-src 'self'",
		ReferrerPolicy:        "same-origin",
		IsDevelopment:         false,
	})
	secureFunc := func() gin.HandlerFunc {
		return func(c *gin.Context) {
			nonce, err := secureMiddleware.ProcessAndReturnNonce(c.Writer, c.Request)
			if err != nil {
				c.Abort()
				return
			}

			c.Set("nonce", nonce)

			if status := c.Writer.Status(); status > 300 && status < 399 {
				c.Abort()
			}
		}
	}()
	r.Use(secureFunc)

	r.ForwardedByClientIP = false // disable ip forward to prevent sproof
	// BUG: gin
	// if !viper.GetBool("proxy.enable") {
	// 	r.ForwardedByClientIP = false // disable ip forward to prevent sproof
	// } else {
	// 	r.ForwardedByClientIP = true
	// 	r.RemoteIPHeaders = []string{viper.GetString("proxy.header")}
	// }

	// CSRF protection
	csrfFunc := func() gin.HandlerFunc {
		return adapter.Wrap(csrf.Protect([]byte(viper.GetString("csrf_secret")), csrf.Path("/"),
			csrf.Secure(viper.GetBool("listen.ssl")), csrf.MaxAge(0), csrf.SameSite(csrf.SameSiteStrictMode)))
	}
	r.Use(csrfFunc())

	// session
	utils.GetSession().Options(sessions.Options{
		Path:     "/",
		MaxAge:   viper.GetInt("session.expire"),
		Secure:   viper.GetBool("listen.ssl"),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// setting
	sn.Skynet.Setting, err = handler.NewSetting()
	if err != nil {
		log.Fatal("Setting init error: ", err)
	}

	// static files
	staticFile := r.Group("/")
	staticFile.Static("/js/main", "./assets/js")
	staticFile.Static("/css/main", "./assets/css")
	staticFile.Static("/fonts/main", "./assets/fonts")
	sn.Skynet.StaticFile = staticFile

	// router & template
	web := r.Group("/")
	t := multitemplate.NewRenderer()
	sn.Skynet.Page = page.NewPage(t, web)
	r.HTMLRender = t

	// api router
	v1 := r.Group("/api")
	sn.Skynet.API = api.NewAPI(v1)

	// plugin
	sn.Skynet.Plugin, err = handler.NewPlugin("plugin")
	if err != nil {
		log.Fatal("Plugin init error: ", err)
	}
	defer sn.Skynet.Plugin.Fini()

	endless.DefaultHammerTime = 1 * time.Second
	server := endless.NewServer(viper.GetString("listen.address"), r)
	server.BeforeBegin = func(add string) {
		log.Info("Running pid ", syscall.Getpid())
	}
	if !viper.GetBool("listen.ssl") {
		server.ListenAndServe()
	} else {
		server.ListenAndServeTLS(viper.GetString("listen.ssl_cert"), viper.GetString("listen.ssl_key"))
	}
}

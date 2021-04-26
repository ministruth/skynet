package cmd

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"os"
	"skynet/db"
	"skynet/handlers"
	"skynet/pages"
	"syscall"

	logrus_stack "github.com/Gurpartap/logrus-stack"
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	adapter "github.com/gwatts/gin-adapter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/unrolled/secure"
)

var logFile *os.File

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run skynet server",
	Run:   run,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func apiVersion(in string) (string, error) {
	return handlers.APIVERSION + in, nil
}

func connectDB(ctx context.Context) {
	switch viper.GetString("database.type") {
	case "sqlite":
		db.InitDB(ctx, &db.DBConfig{
			Type: db.DBType_Sqlite,
			Path: viper.GetString("database.path"),
		})
	default:
		log.Fatalf("Database type %s not supported", viper.GetString("database.type"))
	}
}

func run(cmd *cobra.Command, args []string) {
	// logrus hook
	var err error
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

	// check default settings
	for k, v := range defaultSettings {
		if k[0] == '*' {
			if viper.Get(k[1:]) == v {
				log.Warnf("Setting %v has default value, please modify your config file for safety", k[1:])
			}
		}
	}

	// gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	ctx := context.Background()

	// database
	db.InitRedis(ctx, &db.RedisConfig{
		Address:  viper.GetString("redis.address"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})
	log.WithFields(log.Fields{
		"addr": viper.GetString("redis.address"),
	}).Info("Redis connected")
	connectDB(ctx)
	log.WithFields(log.Fields{
		"path": viper.GetString("database.path"),
	}).Info("Database connected")

	// security
	secureMiddleware := secure.New(secure.Options{
		AllowedHosts:          []string{},
		AllowedHostsAreRegex:  true,
		HostsProxyHeaders:     []string{"X-Forwarded-Hosts"},
		SSLRedirect:           false, // TODO: SSL
		SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
		STSSeconds:            31536000,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: "default-src 'none'; script-src $NONCE; connect-src 'self'; img-src 'self' data:; style-src 'self'; base-uri 'self'; form-action 'self'; font-src 'self'",
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
	r.Use(gin.Recovery())         // recover from panic

	// CSRF protection
	csrfFunc := func() gin.HandlerFunc {
		return adapter.Wrap(csrf.Protect([]byte(viper.GetString("csrf_secret")),
			csrf.Secure(false), csrf.MaxAge(0), csrf.SameSite(csrf.SameSiteStrictMode))) // TODO: SSL
	}
	r.Use(csrfFunc())

	// session
	db.GetSession().Options(sessions.Options{
		Path:     "/",
		MaxAge:   viper.GetInt("session.expire"),
		Secure:   false, // TODO: SSL
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// template
	r.SetFuncMap(template.FuncMap{
		"api": apiVersion,
	})
	r.LoadHTMLGlob("templates/*")

	// static files
	r.Static("/js", "./assets/js")
	r.Static("/css", "./assets/css")
	r.Static("/fonts", "./assets/fonts")

	// router
	web := r.Group("/")
	pages.PageRouter(web)

	// api router
	v1 := r.Group(handlers.APIVERSION)
	handlers.APIRouter(v1)

	server := endless.NewServer(viper.GetString("listen_addr"), r)
	server.BeforeBegin = func(add string) {
		log.Info("Running pid ", syscall.Getpid())
	}
	server.ListenAndServe()
}

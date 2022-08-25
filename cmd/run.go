package cmd

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"skynet/api"
	"skynet/config"
	"skynet/db"
	"skynet/handler"
	"skynet/recaptcha"
	"skynet/security"
	"skynet/sn"
	"skynet/translator"
	"skynet/utils/log"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

var (
	test   bool
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run skynet server",
		Run:   run,
	}
)

func init() {
	runCmd.Flags().BoolVarP(&test, "test", "t", false, "only test config file and db connection")
	rootCmd.AddCommand(runCmd)
}

func run(cmd *cobra.Command, args []string) {
	// logger
	var logWriter []io.Writer
	if viper.GetBool("log.console") {
		logWriter = append(logWriter, os.Stdout)
	}
	if viper.GetBool("log.json") {
		log.SetJSONFormat()
	}
	if viper.GetString("log.file") != "" {
		logFile, err := os.OpenFile(viper.GetString("log_file"), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0755)
		if err != nil {
			log.NewEntry(err).Fatal("Failed to open log file")
		}
		defer logFile.Close()
		logWriter = append(logWriter, logFile)
	}
	if viper.GetBool("log.stack") {
		log.ShowStack()
	}
	if len(logWriter) > 0 {
		log.SetOutput(logWriter...)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	log.New().Info("========== Skynet server start ==========")
	defer log.New().Info("========== Skynet server end ==========")

	// check setting first
	log.New().Infof("config file: %s", conf)
	config.CheckSetting()

	// database
	db.NewRedis()
	db.NewSession()
	db.NewDB()

	// handler
	handler.Init()
	log.New().AddHook(handler.NotificationHook{})

	// recaptcha
	if viper.GetBool("recaptcha.enable") {
		recaptcha.Init()
	}

	// i18n
	translator.New()

	// session
	db.Session.Options(sessions.Options{
		Path:     "/",
		MaxAge:   viper.GetInt("session.expire"),
		Secure:   viper.GetBool("listen.ssl"),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	if !viper.GetBool("debug") {
		if err := db.DeleteSessions(nil); err != nil {
			log.NewEntry(err).Fatal("Failed to clean session")
		}
	}

	if test {
		sn.Running = true // for test purpose
		log.New().Info("Test success")
		return
	}

	// init gin
	var r *gin.Engine
	if len(logWriter) > 0 {
		r = gin.Default()
	} else {
		r = gin.New()
		r.Use(gin.Recovery())
	}

	// security
	r.Use(security.SecureMiddleware())
	r.Use(security.CSRFMiddleware())

	if !viper.GetBool("proxy.enable") {
		r.ForwardedByClientIP = false
		r.SetTrustedProxies(nil)
	} else {
		r.ForwardedByClientIP = true
		r.RemoteIPHeaders = []string{viper.GetString("proxy.header")}
		r.SetTrustedProxies(strings.Split(viper.GetString("proxy.trusted"), ","))
	}

	// plugin auth
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		sp := strings.Split(path, "/")
		if strings.HasPrefix(path, "/_plugin/") {
			if len(sp) >= 3 {
				req, err := api.ParseRequest(c)
				if err != nil {
					log.NewEntry(err).Error("Request handler error")
					c.AbortWithStatus(500)
					return
				}
				ids, err := api.GetMenuPluginID(req)
				if err != nil {
					log.NewEntry(err).Error("Request handler error")
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

	// // gin middleware is in order, so plugin middleware should use callback instead of direct use
	// r.Use(func(c *gin.Context) {
	// 	if errs := sn.Skynet.Plugin.Call(sn.BeforeMiddleware, c); errs != nil {
	// 		for _, v := range errs {
	// 			utils.WithTrace(v).Error(v)
	// 		}
	// 		c.AbortWithStatus(500)
	// 		return
	// 	}
	// 	c.Next()
	// 	sn.Skynet.Plugin.Call(sn.AfterMiddleware, c)
	// })

	// 404
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		sp := strings.Split(path, "/")
		if strings.HasPrefix(path, "/api") {
			// prevent route guess
			c.String(403, "Permission denied")
		} else if path != "/" && !strings.Contains(sp[len(sp)-1], ".") {
			// rewrite for dynamic route
			c.Request.URL.Path = "/"
			r.HandleContext(c)
		} else {
			// static file 404
			c.AbortWithStatus(404)
		}
	})

	// static file
	r.Use(static.Serve("/", static.LocalFile("./assets/", false)))

	// api router
	apiRouter := r.Group("/api")
	api.Init(apiRouter)

	if err := handler.Plugin.LoadPlugin("plugin"); err != nil {
		log.New().Fatal("Plugin init error: ", err)
	}
	defer handler.Plugin.Fini()

	sn.Running = true
	sn.StartTime = time.Now()
	log.New().Info("Running pid ", syscall.Getpid())
	errChan := make(chan error)
	if !viper.GetBool("listen.ssl") {
		go func() { errChan <- tracerr.Wrap(r.Run(viper.GetString("listen.address"))) }()
	} else {
		go func() {
			errChan <- tracerr.Wrap(r.RunTLS(viper.GetString("listen.address"),
				viper.GetString("listen.ssl_cert"), viper.GetString("listen.ssl_key")))
		}()
	}
	select {
	case <-sn.ExitChan:
	case err := <-errChan:
		if err != nil {
			log.NewEntry(err).Error("Failed to start server")
		}
	}
}

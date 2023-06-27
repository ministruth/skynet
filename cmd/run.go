package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MXWXZ/skynet/api"
	"github.com/MXWXZ/skynet/plugin"
	"github.com/MXWXZ/skynet/recaptcha"
	"github.com/MXWXZ/skynet/security"
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/translator"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

var (
	disable_csrf    bool
	persist_session bool
	debug           bool
	runCmd          = &cobra.Command{
		Use:    "run",
		Short:  "Run skynet server",
		Run:    run,
		PreRun: load,
	}
)

func init() {
	runCmd.Flags().BoolVar(&persist_session, "persist-session", false, "persist previous session when initializing")
	runCmd.Flags().BoolVar(&disable_csrf, "disable-csrf", false, "disable csrf protection (for test only)")
	runCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")
	rootCmd.AddCommand(runCmd)
}

func run(cmd *cobra.Command, args []string) {
	// parse param
	if !persist_session {
		if err := sn.Skynet.Session.Delete(nil); err != nil {
			log.NewEntry(err).Fatal("Failed to clean session")
		}
	}

	// session
	sn.Skynet.Session.GetStore().Options(sessions.Options{
		Path:     "/",
		MaxAge:   viper.GetInt("session.expire"),
		Secure:   viper.GetBool("listen.ssl"),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// recaptcha
	if viper.GetBool("recaptcha.enable") {
		var err error
		sn.Skynet.ReCAPTCHA, err = recaptcha.NewReCAPTCHA(
			viper.GetString("recaptcha.secret"),
			time.Duration(viper.GetInt("recaptcha.timeout"))*time.Second)
		if err != nil {
			log.NewEntry(err).Fatal("Failed to init recaptcha")
		}
	}
	// i18n
	translator.New()

	// init gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	if !quiet {
		r.Use(log.GinMiddleware(), gin.Recovery())
	} else {
		// no logger
		r.Use(gin.Recovery())
	}
	sn.Skynet.Engine = r

	// security
	r.Use(security.SecureMiddleware(debug))
	if disable_csrf {
		log.New().Warn("CSRF protection is disabled, enable it when put into production")
	} else {
		r.Use(security.CSRFMiddleware())
	}

	// proxy
	if !viper.GetBool("proxy.enable") {
		r.ForwardedByClientIP = false
		r.SetTrustedProxies(nil)
	} else {
		r.ForwardedByClientIP = true
		r.RemoteIPHeaders = []string{viper.GetString("proxy.header")}
		r.SetTrustedProxies(strings.Split(viper.GetString("proxy.trusted"), ","))
	}

	// 404
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		sp := strings.Split(path, "/")
		if path != "/" && !strings.Contains(sp[len(sp)-1], ".") {
			// rewrite for dynamic route
			c.Request.URL.Path = "/"
			r.HandleContext(c)
		} else {
			c.AbortWithStatus(404)
		}
	})

	// api router
	apiRouter := r.Group("/api")
	api.Init(apiRouter)

	// init plugin
	sn.Skynet.Plugin = plugin.NewPlugin(pluginDir)

	// start skynet
	sn.Skynet.StartTime = time.Now()
	log.New().WithField("pid", syscall.Getpid()).Info("Running pid ", syscall.Getpid())
	errChan := make(chan error)
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt)
	srv := &http.Server{
		Addr:    viper.GetString("listen.address"),
		Handler: r,
	}
	if !viper.GetBool("listen.ssl") {
		go func() { errChan <- tracerr.Wrap(srv.ListenAndServe()) }()
	} else {
		go func() {
			errChan <- tracerr.Wrap(srv.ListenAndServeTLS(viper.GetString("listen.ssl_cert"), viper.GetString("listen.ssl_key")))
		}()
	}
	log.New().Infof("Listening on %v...", viper.GetString("listen.address"))

	// gracefully exit
	quit := func() {
		log.New().Info("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.NewEntry(err).Fatal("Error during shutdown")
		}
		sn.Skynet.Plugin.Unload()
	}
	select {
	case <-intChan:
		quit()
	case <-sn.Skynet.ExitChan:
		quit()
	case err := <-errChan:
		if err != nil {
			log.NewEntry(err).Error("Failed to start server")
		}
	}
}

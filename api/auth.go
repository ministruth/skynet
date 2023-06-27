package api

import (
	"time"

	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/security"
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func APISignIn(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Username  string `json:"username" binding:"required,max=32"`
		Password  string `json:"password" binding:"required"`
		Remember  bool   `json:"remember"`
		Recaptcha string `json:"g-recaptcha-response"`
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	logger := req.Logger.WithFields(log.F{
		"username": param.Username,
		"remember": param.Remember,
	})

	if viper.GetBool("recaptcha.enable") {
		if err := sn.Skynet.ReCAPTCHA.Verify(param.Recaptcha, req.Context.ClientIP()); err != nil {
			log.WrapEntry(logger, err).Debug("Failed to check recaptcha")
			return &sn.Response{Code: sn.CodeRecaptchaInvalid}, nil
		}
	}

	user, res, err := sn.Skynet.User.CheckPass(param.Username, param.Password)
	if err != nil {
		return nil, err
	}

	switch res {
	case 0: // signin
		now := time.Now()
		if err := sn.Skynet.User.Update([]string{"last_login", "last_ip"}, &sn.User{
			GeneralFields: sn.GeneralFields{ID: user.ID},
			LastLogin:     now.UnixMilli(),
			LastIP:        req.Context.ClientIP(),
		}); err != nil {
			return nil, err
		}

		session, err := db.GetCTXSession(req.Context)
		if err != nil {
			return nil, err
		}
		data := &sn.SessionData{
			ID:   user.ID,
			Time: now.Unix(),
		}
		data.SaveSession(session)
		if param.Remember {
			session.Options.MaxAge = viper.GetInt("session.remember")
		} else {
			session.Options.MaxAge = viper.GetInt("session.expire")
		}
		if err := db.SaveCTXSession(req.Context); err != nil {
			return nil, err
		}

		success(logger.WithField("uid", user.ID), "User signin")
		return sn.ResponseOK, nil
	default: // invalid
		logger.Warn("Invalid username or password")
		return &sn.Response{Code: sn.CodeUserInvalid}, nil
	}
}

func APIPing(req *sn.Request) (*sn.Response, error) {
	return sn.ResponseOK, nil
}

func APISignOut(req *sn.Request) (*sn.Response, error) {
	session, err := db.GetCTXSession(req.Context)
	if err != nil {
		return nil, err
	}
	session.Options.MaxAge = -1
	if err = db.SaveCTXSession(req.Context); err != nil {
		return nil, err
	}
	return sn.ResponseOK, nil
}

func APIGetAccess(req *sn.Request) (*sn.Response, error) {
	perm := make(map[string]sn.UserPerm)
	if req.ID != uuid.Nil {
		for _, v := range req.Perm {
			perm[v.Name] = v.Perm
		}
		return &sn.Response{Data: gin.H{
			"signin":     true,
			"id":         req.ID,
			"permission": perm,
		}}, nil
	} else {
		perm["guest"] = sn.PermAll
		return &sn.Response{Data: gin.H{
			"signin":     false,
			"permission": perm,
		}}, nil
	}
}

func APIGetCSRFToken(req *sn.Request) (*sn.Response, error) {
	token, err := security.NewCSRFToken()
	if err != nil {
		return nil, err
	}

	req.Context.SetCookie(security.CSRF_COOKIE, token,
		viper.GetInt("csrf.expire"), "/", "", viper.GetBool("listen.ssl"), false) // http_only must not be set
	return sn.ResponseOK, nil
}

func APIShutdown(req *sn.Request) (*sn.Response, error) {
	req.Logger.Warn("Manually shutdown skynet")
	go func() {
		// return to user first
		time.Sleep(time.Second * 2)
		sn.Skynet.ExitChan <- true
	}()
	return sn.ResponseOK, nil
}

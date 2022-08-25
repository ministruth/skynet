package api

import (
	"skynet/db"
	"skynet/handler"
	"skynet/recaptcha"
	"skynet/security"
	"skynet/sn"
	"skynet/utils/log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func APISignIn(req *Request) (*Response, error) {
	type Param struct {
		Username  string `json:"username" binding:"required,max=32"`
		Password  string `json:"password" binding:"required"`
		Remember  bool   `json:"remember"`
		Recaptcha string `json:"g-recaptcha-response"`
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return rspParamInvalid, err
	}
	logger := req.Logger.WithFields(log.F{
		"username": param.Username,
		"remember": param.Remember,
	})

	if viper.GetBool("recaptcha.enable") {
		if err := recaptcha.ReCAPTCHA.VerifyCAPTCHA(param.Recaptcha, req.Context.ClientIP()); err != nil {
			log.MergeEntry(logger, log.NewEntry(err)).Debug("Failed to check recaptcha")
			return &Response{Code: CodeInvalidRecaptcha}, nil
		}
	}

	user, res, err := handler.User.CheckPass(param.Username, param.Password)
	if err != nil {
		return nil, err
	}

	switch res {
	case 0: // signin
		now := time.Now()
		if err := handler.User.Update(user.ID, "", "", nil, &now, req.Context.ClientIP()); err != nil {
			return nil, err
		}

		session, err := db.GetCTXSession(req.Context)
		if err != nil {
			return nil, err
		}
		data := &db.SessionData{
			ID: user.ID,
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
		return rspOK, nil
	default: // invalid
		logger.Warn("Invalid username or password")
		return &Response{Code: CodeInvalidUserOrPass}, nil
	}
}

func APIGetAccess(req *Request) (*Response, error) {
	perm := make(map[string]db.UserPerm)
	if req.ID != uuid.Nil {
		for _, v := range req.Perm {
			perm[v.Name] = v.Perm
		}
		return &Response{Data: gin.H{
			"signin":     true,
			"permission": perm,
		}}, nil
	} else {
		return &Response{Data: gin.H{
			"signin":     false,
			"permission": perm,
		}}, nil
	}
}

func APISignOut(req *Request) (*Response, error) {
	session, err := db.GetCTXSession(req.Context)
	if err != nil {
		return nil, err
	}
	session.Options.MaxAge = -1
	if err = db.SaveCTXSession(req.Context); err != nil {
		return nil, err
	}
	success(req.Logger, "User signout")
	return rspOK, nil
}

func APIReload(req *Request) (*Response, error) {
	sn.Running = false
	req.Logger.Warn("Restart skynet")
	go func() {
		time.Sleep(time.Second * 2)
		log.New().Warn("Restart triggered")
		sn.ExitChan <- true
	}()
	return rspOK, nil
}

func APIGetCSRFToken(req *Request) (*Response, error) {
	token, err := security.NewCSRFToken()
	if err != nil {
		return nil, err
	}
	return &Response{Data: token}, nil
}

func APIPing(req *Request) (*Response, error) {
	return rspOK, nil
}

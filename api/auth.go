package api

import (
	"skynet/sn"
	"skynet/sn/impl"
	"skynet/sn/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

/**
 * @api {post} /signin Signin
 * @apiName Signin
 * @apiVersion 1.0.0
 * @apiGroup Auth
 * @apiDescription Signin with username and password.
 * @apiBody {string{..32}} username login username
 * @apiBody {string} password login password
 * @apiBody {bool} [remember=false] remember login
 * @apiBody {string} [g-recaptcha-response] recaptcha response
 * @apiSuccess {int32} code 0 for success, 1 for invalid user, 2 for invalid recaptcha
 * @apiSuccess {string} msg return message
 * @apiPermission guest
 * @apiUse 400
 * @apiUse 500
 */
func APISignIn(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		Username  string `json:"username" binding:"required,max=32"`
		Password  string `json:"password" binding:"required"`
		Remember  bool   `json:"remember"`
		Recaptcha string `json:"g-recaptcha-response"`
	}
	var param Param
	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	logf := wrap(c, id, log.Fields{
		"username": param.Username,
		"remember": param.Remember,
	})

	if viper.GetBool("recaptcha.enable") {
		if err := utils.VerifyCAPTCHA(param.Recaptcha, utils.GetIP(c)); err != nil {
			utils.WithLogTrace(logf, err).Warn(err)
			response(c, CodeInvalidRecaptcha)
			return 0, nil
		}
	}

	user, res, err := sn.Skynet.User.CheckPass(param.Username, param.Password)
	if err != nil {
		return 500, err
	}

	switch res {
	case 0: // signin
		now := time.Now()
		if err := sn.Skynet.User.Update(user.ID, "", "", nil, &now, utils.GetIP(c)); err != nil {
			return 500, err
		}

		session, err := impl.GetCTXSession(c)
		if err != nil {
			return 500, err
		}
		data := &impl.SessionData{
			ID: user.ID,
		}
		data.SaveSession(session)
		if param.Remember {
			session.Options.MaxAge = viper.GetInt("session.remember")
		} else {
			session.Options.MaxAge = viper.GetInt("session.expire")
		}
		if err := impl.SaveCTXSession(c); err != nil {
			return 500, err
		}

		success(logf.WithField("uid", user.ID), "User signin")
		response(c, CodeOK)
	default: // invalid
		logf.Warn("Invalid username or password")
		response(c, CodeInvalidUserOrPass)
	}
	return 0, nil
}

func APIGetAccess(c *gin.Context, id uuid.UUID) (int, error) {
	perm := make(map[string]sn.UserPerm)
	if id != uuid.Nil {
		p, err := impl.GetPerm(id)
		if err != nil {
			return 500, err
		}
		for _, v := range p {
			perm[v.Name] = v.Perm
		}
		responseData(c, gin.H{
			"signin":     true,
			"permission": perm,
		})
	} else {
		responseData(c, gin.H{
			"signin":     false,
			"permission": perm,
		})
	}
	return 0, nil
}

func APISignOut(c *gin.Context, id uuid.UUID) (int, error) {
	logf := wrap(c, id, nil)
	session, err := impl.GetCTXSession(c)
	if err != nil {
		return 500, err
	}
	session.Options.MaxAge = -1
	if err = impl.SaveCTXSession(c); err != nil {
		return 500, err
	}
	success(logf, "User signout")
	response(c, CodeOK)
	return 0, nil
}

func APIReload(c *gin.Context, id uuid.UUID) (int, error) {
	sn.Skynet.Running = false
	logf := wrap(c, id, nil)
	logf.Warn("Restart skynet")
	response(c, CodeOK)
	go func() {
		time.Sleep(time.Second * 2)
		utils.Restart()
	}()
	return 0, nil
}

func APIGetCSRFToken(c *gin.Context, id uuid.UUID) (int, error) {
	token, err := impl.NewCSRFToken()
	if err != nil {
		return 500, err
	}
	responseData(c, token)
	return 0, nil
}

func APIPing(c *gin.Context, id uuid.UUID) (int, error) {
	if sn.Skynet.Running {
		response(c, CodeOK)
	} else {
		response(c, CodeRestarting)
	}
	return 0, nil
}

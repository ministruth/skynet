package recaptcha

import (
	"time"

	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
	"gopkg.in/ezzarghili/recaptcha-go.v4"
)

var (
	ErrReCAPTCHANotInit = tracerr.New("reCAPTCHA not init")
)

type ReCAPTCHAImpl struct {
	recaptcha.ReCAPTCHA
	inited bool
}

func NewReCAPTCHA(secret string, timeout time.Duration) (sn.ReCAPTCHA, error) {
	var err error
	ret := new(ReCAPTCHAImpl)
	ret.ReCAPTCHA, err = recaptcha.NewReCAPTCHA(secret, recaptcha.V2, timeout)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	if viper.GetBool("recaptcha.cnmirror") {
		ret.ReCAPTCHALink = "https://www.recaptcha.net/recaptcha/api/siteverify"
	}
	ret.inited = true
	log.New().WithFields(log.F{
		"link":    ret.ReCAPTCHALink,
		"timeout": timeout,
	}).Debug("ReCAPTCHA initialized")
	return ret, nil
}

// VerifyCAPTCHA verify reCAPTCHA based on response and ip.
func (r *ReCAPTCHAImpl) Verify(response string, ip string) error {
	if !r.inited {
		return ErrReCAPTCHANotInit
	}
	return tracerr.Wrap(r.VerifyWithOptions(response, recaptcha.VerifyOption{
		RemoteIP: ip,
	}))
}

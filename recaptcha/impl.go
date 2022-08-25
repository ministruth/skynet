package recaptcha

import (
	"time"

	"github.com/ztrue/tracerr"
	"gopkg.in/ezzarghili/recaptcha-go.v4"
)

var (
	ErrReCAPTCHANotInit = tracerr.New("reCAPTCHA not init")
)

type ReCAPTCHAInstance struct {
	recaptcha.ReCAPTCHA
	inited bool
}

func Instance(secret string, cnmirror bool, timeout int) (*ReCAPTCHAInstance, error) {
	ret := new(ReCAPTCHAInstance)
	tmp, err := recaptcha.NewReCAPTCHA(secret, recaptcha.V2, time.Duration(timeout)*time.Second)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	ret.ReCAPTCHA = tmp
	if cnmirror {
		ret.ReCAPTCHALink = "https://www.recaptcha.net/recaptcha/api/siteverify"
	}
	ret.inited = true
	return ret, nil
}

// VerifyCAPTCHA verify reCAPTCHA based on response and ip.
func (r *ReCAPTCHAInstance) VerifyCAPTCHA(response string, ip string) error {
	if !r.inited {
		return ErrReCAPTCHANotInit
	}
	return tracerr.Wrap(r.VerifyWithOptions(response, recaptcha.VerifyOption{
		RemoteIP: ip,
	}))
}

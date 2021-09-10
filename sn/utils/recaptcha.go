package utils

import (
	"time"

	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
	"gopkg.in/ezzarghili/recaptcha-go.v4"
)

var reInstance *recaptcha.ReCAPTCHA

var (
	ErrReCAPTCHANotInit = tracerr.New("reCAPTCHA not init")
)

// NewReCAPTCHA Init new reCAPTCHA instance based on secret.
func NewReCAPTCHA(secret string) error {
	tmp, err := recaptcha.NewReCAPTCHA(secret, recaptcha.V2, 10*time.Second)
	if err != nil {
		return tracerr.Wrap(err)
	}
	reInstance = &tmp
	if viper.GetBool("recaptcha.cnmirror") {
		reInstance.ReCAPTCHALink = "https://www.recaptcha.net/recaptcha/api/siteverify"
	}
	return nil
}

// VerifyCAPTCHA verify reCAPTCHA based on response and ip.
func VerifyCAPTCHA(response string, ip string) error {
	if reInstance == nil {
		return ErrReCAPTCHANotInit
	}
	return tracerr.Wrap(reInstance.VerifyWithOptions(response, recaptcha.VerifyOption{
		RemoteIP: ip,
	}))
}

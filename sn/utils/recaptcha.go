package utils

import (
	"time"

	"github.com/spf13/viper"
	"gopkg.in/ezzarghili/recaptcha-go.v4"
)

var reInstance *recaptcha.ReCAPTCHA

func NewReCAPTCHA(secret string) error {
	tmp, err := recaptcha.NewReCAPTCHA(secret, recaptcha.V2, 10*time.Second)
	if err == nil {
		reInstance = &tmp
	}
	if viper.GetBool("recaptcha.cnmirror") {
		tmp.ReCAPTCHALink = "https://www.recaptcha.net/recaptcha/api/siteverify"
	}
	return err
}

func VerifyCAPTCHA(response string, ip string) error {
	if reInstance == nil {
		panic("reCAPTCHA not init")
	}
	return reInstance.VerifyWithOptions(response, recaptcha.VerifyOption{
		RemoteIP: ip,
	})
}

package recaptcha

import (
	"skynet/utils/log"

	"github.com/spf13/viper"
)

var ReCAPTCHA *ReCAPTCHAInstance

// Init init new skynet reCAPTCHA instance.
func Init() {
	var err error
	ReCAPTCHA, err = Instance(viper.GetString("recaptcha.secret"),
		viper.GetBool("recaptcha.cnmirror"), viper.GetInt("recaptcha.timeout"))
	if err != nil {
		log.NewEntry(err).Fatal("Failed to init recaptcha")
	}
	log.New().WithFields(log.F{
		"link": ReCAPTCHA.ReCAPTCHALink,
	}).Debug("ReCAPTCHA initialized")
}

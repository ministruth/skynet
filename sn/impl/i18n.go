package impl

import (
	"skynet/sn/utils"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/ztrue/tracerr"
)

func TranslateString(t *i18n.Localizer, s string) string {
	ret, err := t.Localize(&i18n.LocalizeConfig{
		MessageID: s,
	})
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Warn(err)
		return s
	}
	return ret
}

func TranslateTpl(t *i18n.Localizer, s string, tpl interface{}) string {
	ret, err := t.Localize(&i18n.LocalizeConfig{
		MessageID:    s,
		TemplateData: tpl,
	})
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Warn(err)
		return s
	}
	return ret
}

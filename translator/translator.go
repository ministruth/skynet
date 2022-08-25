package translator

import (
	"embed"
	"io/fs"
	"skynet/utils/log"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/ztrue/tracerr"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

var Translator *i18n.Bundle // skynet i18n

//go:embed translate/*.yml
var i18nFiles embed.FS

func NewLocalizer(lang ...string) *i18n.Localizer {
	return i18n.NewLocalizer(Translator, lang...)
}

func New() {
	Translator = i18n.NewBundle(language.English)
	Translator.RegisterUnmarshalFunc("yml", yaml.Unmarshal)
	err := fs.WalkDir(i18nFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return tracerr.Wrap(err)
		}
		if !d.IsDir() {
			b, err := i18nFiles.ReadFile("translate/" + d.Name())
			if err != nil {
				return tracerr.Wrap(err)
			}
			Translator.MustParseMessageFileBytes(b, d.Name())
			log.New().Debugf("Language %v loaded", d.Name())
		}
		return nil
	})
	if err != nil {
		log.NewEntry(err).Fatal("Failed to read localize files")
	}
}

// TranslateString translate string to target language.
// If error happened, return untranslated string.
func TranslateString(t *i18n.Localizer, s string) string {
	ret, err := t.Localize(&i18n.LocalizeConfig{
		MessageID: s,
	})
	if err != nil {
		log.NewEntry(tracerr.Wrap(err)).WithField("messageID", s).
			Warn("Failed to localize string")
		return s
	}
	return ret
}

// TranslateTpl translate string and template to target language.
// If error happened, return untranslated string.
func TranslateTpl(t *i18n.Localizer, s string, tpl any) string {
	ret, err := t.Localize(&i18n.LocalizeConfig{
		MessageID:    s,
		TemplateData: tpl,
	})
	if err != nil {
		log.NewEntry(tracerr.Wrap(err)).
			WithFields(log.F{"messageID": s, "template": tpl}).
			Warn("Failed to update setting")
		return s
	}
	return ret
}

package sn

import (
	"testing"

	"github.com/MXWXZ/skynet/translator"
	"github.com/stretchr/testify/assert"
)

func TestResponseCode(t *testing.T) {
	translator.New()
	tr := translator.NewLocalizer("en-US")
	t.Run("Test Code", func(t *testing.T) {
		for i := 0; i < int(CodeMax); i++ {
			str := ResponseCode(i).GetMsg()
			assert.NotEqual(t, str, translator.TranslateString(tr, str))
		}
	})
}

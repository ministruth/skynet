package api

import (
	"skynet/sn/impl"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type RspCode int32

const (
	CodeOK RspCode = iota
	CodeInvalidUserOrPass
	CodeUsernameExist
	CodeInvalidRecaptcha
	CodeRestarting
	CodeGroupNotExist
	CodeGroupNameExist
	CodeRootNotAllowedUpdate
	CodeRootNotAllowedDelete
	CodePluginNotExist
	CodePluginFormatError
	CodePluginExist

	CodeMax
)

var RspMsg = [CodeMax]string{
	"response.success",
	"response.user.invalid",
	"response.user.exist",
	"response.recaptcha.invalid",
	"response.restart",
	"response.group.notexist",
	"response.group.exist",
	"response.group.rootupdate",
	"response.group.rootdelete",
	"response.plugin.notexist",
	"response.plugin.formaterror",
	"response.plugin.exist",
}

func (d RspCode) String(c *gin.Context) string {
	t := c.MustGet("translator").(*i18n.Localizer)
	return impl.TranslateString(t, RspMsg[d])
}

func response(c *gin.Context, code RspCode) {
	c.JSON(200, gin.H{"code": code, "msg": code.String(c)})
}

func responseData(c *gin.Context, data any) {
	c.JSON(200, gin.H{"code": CodeOK, "msg": CodeOK.String(c), "data": data})
}

func responsePage(c *gin.Context, data any, total any) {
	c.JSON(200, gin.H{"code": CodeOK, "msg": CodeOK.String(c), "data": data, "total": total})
}

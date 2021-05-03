package sn

import "github.com/gin-gonic/gin"

type SNGlobal struct {
	PageRouter *gin.RouterGroup
	APIRouter  *gin.RouterGroup
	Page       SNPage
	Plugin     SNPlugin
	Setting    SNSetting
	User       SNUser
	DB         SNDB
	Redis      SNDB
	Session    SNDB
	ShareData  map[string]interface{}
}

var Skynet SNGlobal

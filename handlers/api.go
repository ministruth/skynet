package handlers

import (
	"skynet/utils"

	"github.com/gin-gonic/gin"
)

const APIVERSION = "/v1"

func APIRouter(r *gin.RouterGroup) {
	r.POST("/signin", SignIn)
	r.GET("/signout", utils.NeedSignIn(SignOut, false))
}

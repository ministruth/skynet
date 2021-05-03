package api

import (
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/gin-gonic/gin"
)

const APIVERSION = "/v1"

func APIRouter(r *gin.RouterGroup) {
	// TODO: Add new api router
	r.POST("/signin", APISignIn)

	r.GET("/signout", utils.NeedSignIn(APISignOut, false))
	r.PATCH("/user", utils.NeedSignIn(APIEditUser, false))

	r.POST("/user", utils.NeedAdmin(APIAddUser, false))
	r.DELETE("/user", utils.NeedAdmin(APIDeleteUser, false))
	r.GET("/reload", utils.NeedAdmin(APIReload, false))
	r.PATCH("/plugin", utils.NeedAdmin(APIEditPlugin, false))
}

func APIReload(c *gin.Context, u *sn.Users) {
	c.JSON(200, gin.H{"code": 0, "msg": "Restarting skynet..."})
	go func() {
		time.Sleep(time.Second * 2)
		utils.Restart()
	}()
}

package page

import (
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type userParam struct {
	sn.Users
	Online bool
}

func PageUser(c *gin.Context, u *sn.Users) {
	users, err := sn.Skynet.User.GetUser()
	if err != nil {
		log.Error(err)
		c.AbortWithStatus(500)
		return
	}

	param := make([]userParam, len(users))
	for i := range users {
		s, err := utils.FindSessionsByID(int(users[i].ID))
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(500)
			return
		}
		param[i].Users = users[i]
		param[i].Online = len(s) != 0
	}
	sn.Skynet.Page.Render(c, "user", "Skynet | User", "User", "/user", u, gin.H{
		"users": param,
		"_path": append(sn.SNDefaultPath, &sn.SNPageItem{
			Name:   "User",
			Active: true,
		}),
	})
}

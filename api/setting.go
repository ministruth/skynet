package api

import (
	"fmt"
	"skynet/sn"
	"skynet/sn/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

type updateNavbarParam struct {
	Order []string `json:"order"`
}

func APIUpdateNavbar(c *gin.Context, u *sn.User) (int, error) {
	var param updateNavbarParam
	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	logf := log.WithFields(log.Fields{
		"ip": utils.GetIP(c),
	})

	navbar := sn.Skynet.Page.GetNav()
	verify := func(id string) *sn.SNNavItem {
		var dfs func(s *sn.SNNavItem) *sn.SNNavItem
		dfs = func(s *sn.SNNavItem) *sn.SNNavItem {
			if s.ID == id {
				return s
			}
			for _, v := range s.Child {
				if ret := dfs(v); ret != nil {
					return ret
				}
			}
			return nil
		}
		for _, v := range navbar {
			if ret := dfs(v); ret != nil {
				return ret
			}
		}
		return nil
	}
	for i, v := range param.Order {
		if rec := verify(v); rec != nil {
			err := sn.Skynet.Setting.Set(fmt.Sprintf("navbar_priority_%v", rec.ID), strconv.Itoa(i))
			if err != nil {
				return 500, err
			}
			rec.Priority = uint16(i)
		}
	}
	sn.SNNavSort(navbar).Sort()
	logf.Info("Sort navbar success")
	c.JSON(200, gin.H{"code": 0, "msg": "Sort navbar success"})
	return 0, nil
}

package api

import (
	"skynet/handler"
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func APIGetPlugin(c *gin.Context, u *sn.User) (int, error) {
	var param paginationParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		return 400, err
	}

	rec := sn.Skynet.Plugin.GetAll().(map[uuid.UUID]*handler.PluginLoad)
	count := sn.Skynet.Plugin.Count()
	if len(rec) > 0 && (param.Page-1)*param.Size < len(rec) {
		var ret []*handler.PluginLoad
		for i := range rec {
			ret = append(ret, rec[i])
		}
		c.JSON(200, gin.H{"code": 0, "msg": "Get all plugin success", "data": ret[(param.Page-1)*param.Size : utils.IntMin(param.Page*param.Size, len(ret))],
			"total": count})
	} else {
		c.JSON(200, gin.H{"code": 0, "msg": "Get all plugin success", "data": []*handler.PluginLoad{}, "total": count})
	}
	return 0, nil
}

type updatePluginParam struct {
	Enable bool `form:"enable"`
}

func APIUpdatePlugin(c *gin.Context, u *sn.User) (int, error) {
	var param updatePluginParam
	err := c.ShouldBind(&param)
	if err != nil {
		return 400, err
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return 400, err
	}
	logf := log.WithFields(log.Fields{
		"ip":     c.ClientIP(),
		"id":     u.ID,
		"plugin": id,
	})

	if param.Enable {
		err = sn.Skynet.Plugin.Enable(id)
		if err != nil {
			logf.Warn("Enable plugin fail")
			c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
			return 0, nil
		}
		logf.Info("Enable plugin success")
		c.JSON(200, gin.H{"code": 0, "msg": "Enable plugin success"})
	} else {
		err = sn.Skynet.Plugin.Disable(id)
		if err != nil {
			logf.Warn("Disable plugin fail")
			c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
			return 0, nil
		}
		logf.Info("Disable plugin success")
		c.JSON(200, gin.H{"code": 0, "msg": "Disable plugin success, reloading"})
		go func() {
			time.Sleep(time.Second * 2)
			utils.Restart()
		}()
	}
	return 0, nil
}

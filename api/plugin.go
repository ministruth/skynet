package api

import (
	"errors"
	"skynet/handler"
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vincent-petithory/dataurl"
	"github.com/ztrue/tracerr"
)

func APIGetPlugin(c *gin.Context, u *sn.User) (int, error) {
	var param paginationParam
	if err := tracerr.Wrap(c.ShouldBindQuery(&param)); err != nil {
		return 400, err
	}

	rec := sn.Skynet.Plugin.GetAll().(*handler.PluginMap)
	count := sn.Skynet.Plugin.Count()
	if count > 0 && (param.Page-1)*param.Size < count {
		c.JSON(200, gin.H{"code": 0, "msg": "Get all plugin success", "data": rec.Values()[(param.Page-1)*param.Size : utils.IntMin(param.Page*param.Size, count)],
			"total": count})
	} else {
		c.JSON(200, gin.H{"code": 0, "msg": "Get all plugin success", "data": []*handler.PluginLoad{}, "total": count})
	}
	return 0, nil
}

type addPluginParam struct {
	File string `json:"file"`
}

func APIAddPlugin(c *gin.Context, u *sn.User) (int, error) {
	var param addPluginParam
	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	logf := log.WithFields(log.Fields{
		"ip": utils.GetIP(c),
		"id": u.ID,
	})

	f, err := dataurl.DecodeString(param.File)
	if err != nil {
		return 500, tracerr.Wrap(err)
	}
	if err := sn.Skynet.Plugin.New(f.Data); err != nil {
		if errors.Is(err, handler.ErrPluginInvalid) {
			c.JSON(200, gin.H{"code": 1, "msg": "Plugin package format error"})
			return 0, nil
		}
		if errors.Is(err, handler.ErrPluginExists) {
			c.JSON(200, gin.H{"code": 2, "msg": "Plugin already exists"})
			return 0, nil
		}
		return 500, err
	}
	logf.Info("Add plugin success")
	c.JSON(200, gin.H{"code": 0, "msg": "Add plugin success"})
	return 0, nil
}

func APIDeletePlugin(c *gin.Context, u *sn.User) (int, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return 400, tracerr.Wrap(err)
	}
	logf := log.WithFields(log.Fields{
		"ip":     utils.GetIP(c),
		"id":     u.ID,
		"plugin": id,
	})

	if err := sn.Skynet.Plugin.Delete(id); err != nil {
		return 500, err
	}
	logf.Info("Delete plugin success")
	c.JSON(200, gin.H{"code": 0, "msg": "Delete plugin success, reloading"})
	go func() {
		time.Sleep(time.Second * 2)
		utils.Restart()
	}()
	return 0, nil
}

type updatePluginParam struct {
	File   string `json:"file"`
	Enable bool   `json:"enable"`
}

func APIUpdatePlugin(c *gin.Context, u *sn.User) (int, error) {
	var param updatePluginParam
	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return 400, tracerr.Wrap(err)
	}
	logf := log.WithFields(log.Fields{
		"ip":     utils.GetIP(c),
		"id":     u.ID,
		"plugin": id,
	})

	if param.File != "" {
		f, err := dataurl.DecodeString(param.File)
		if err != nil {
			return 500, tracerr.Wrap(err)
		}
		if err := sn.Skynet.Plugin.Update(id, f.Data); err != nil {
			return 500, err
		}
		logf.Info("Update plugin success")
		c.JSON(200, gin.H{"code": 0, "msg": "Update plugin success, reloading"})
		go func() {
			time.Sleep(time.Second * 2)
			utils.Restart()
		}()
		return 0, nil
	}

	if param.Enable {
		if err := sn.Skynet.Plugin.Enable(id); err != nil {
			return 500, err
		}
		logf.Info("Enable plugin success")
		c.JSON(200, gin.H{"code": 0, "msg": "Enable plugin success"})
	} else {
		if err := sn.Skynet.Plugin.Disable(id); err != nil {
			return 500, err
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

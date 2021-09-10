package main

import (
	plugins "skynet/plugin"
	"skynet/plugin/task/shared"
	"skynet/sn"
	"skynet/sn/utils"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

func APIGetAllTask(c *gin.Context, u *sn.User) (int, error) {
	var param plugins.PaginationParam
	if err := tracerr.Wrap(c.ShouldBindQuery(&param)); err != nil {
		return 400, err
	}

	rec, err := pluginAPI.GetAll(&sn.SNCondition{
		Order:  []interface{}{"id " + param.Order},
		Limit:  param.Size,
		Offset: (param.Page - 1) * param.Size,
	})
	if err != nil {
		return 500, err
	}
	count, err := pluginAPI.Count()
	if err != nil {
		return 500, err
	}
	for i := range rec {
		s := strings.Split(strings.TrimSpace(rec[i].Output), "\n")
		rec[i].Output = s[len(s)-1]
	}
	c.JSON(200, gin.H{"code": 0, "msg": "Get all task success", "data": rec, "total": count})
	return 0, nil
}

func APIDeleteInactiveTask(c *gin.Context, u *sn.User) (int, error) {
	logf := log.WithFields(defaultField).WithFields(log.Fields{
		"ip": utils.GetIP(c),
	})
	err := tracerr.Wrap(utils.GetDB().Where("status <> ? and status <> ?", shared.TaskNotStart, shared.TaskRunning).
		Delete(&shared.PluginTask{}).Error)
	if err != nil {
		return 500, err
	}
	logf.Info("Delete inactive task success")
	c.JSON(200, gin.H{"code": 0, "msg": "Delete inactive task success"})
	return 0, nil
}

type getTaskParam struct {
	ID int `uri:"id" binding:"required,min=1"`
}

func APIGetTask(c *gin.Context, u *sn.User) (int, error) {
	var param getTaskParam
	if err := tracerr.Wrap(c.ShouldBindUri(&param)); err != nil {
		return 400, err
	}

	rec, err := pluginAPI.Get(param.ID)
	if err != nil {
		return 500, err
	}
	c.JSON(200, gin.H{"code": 0, "msg": "Get task success", "data": rec})
	return 0, nil
}

type killTaskParam struct {
	ID int `uri:"id" binding:"required,min=1"`
}

func APIKillTask(c *gin.Context, u *sn.User) (int, error) {
	var param killTaskParam
	if err := tracerr.Wrap(c.ShouldBindUri(&param)); err != nil {
		return 400, err
	}
	logf := log.WithFields(defaultField).WithFields(log.Fields{
		"ip": utils.GetIP(c),
		"id": param.ID,
	})

	err := pluginAPI.CancelByUser(param.ID, "Task killed by user")
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return 0, nil
	}
	logf.Info("Kill task success")
	c.JSON(200, gin.H{"code": 0, "msg": "Kill task success"})
	return 0, nil
}

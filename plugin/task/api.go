package main

import (
	"skynet/plugin/task/shared"
	"skynet/sn"
	"skynet/sn/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

type getAllTaskParam struct {
	Page int `form:"page" binding:"min=0"`
	Size int `form:"size" binding:"oneof=5 10 20 50"`
}

func APIGetAllTask(c *gin.Context, u *sn.Users) (int, error) {
	var param getAllTaskParam
	err := c.ShouldBindQuery(&param)
	if err != nil {
		return 400, err
	}
	if param.Page == 0 {
		param.Page = 1
	}

	rec, err := pluginAPI.GetAll([]interface{}{"id desc"}, param.Size, (param.Page-1)*param.Size, nil)
	if err != nil {
		return 500, err
	}
	for i := range rec {
		s := strings.Split(strings.TrimSpace(rec[i].Output), "\n")
		rec[i].Output = s[len(s)-1]
	}
	c.JSON(200, gin.H{"code": 0, "msg": "Get all task success", "data": rec})
	return 0, nil
}

func APIDeleteInactiveTask(c *gin.Context, u *sn.Users) (int, error) {
	err := utils.GetDB().Where("status <> ? and status <> ?", shared.TaskNotStart, shared.TaskRunning).
		Delete(&shared.PluginTasks{}).Error
	if err != nil {
		return 500, err
	}
	c.JSON(200, gin.H{"code": 0, "msg": "Delete inactive task success"})
	return 0, nil
}

type getTaskParam struct {
	ID int `uri:"id" binding:"required,min=1"`
}

func APIGetTask(c *gin.Context, u *sn.Users) (int, error) {
	var param getTaskParam
	err := c.ShouldBindUri(&param)
	if err != nil {
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

func APIKillTask(c *gin.Context, u *sn.Users) (int, error) {
	var param killTaskParam
	err := c.ShouldBindUri(&param)
	if err != nil {
		return 400, err
	}

	pluginAPI.Cancel(param.ID)
	err = pluginAPI.UpdateStatus(param.ID, shared.TaskStop)
	if err != nil {
		return 500, err
	}
	c.JSON(200, gin.H{"code": 0, "msg": "Kill task success"})
	return 0, nil
}

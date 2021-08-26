package main

import (
	"fmt"
	plugins "skynet/plugin"
	"skynet/plugin/task/shared"
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var Instance = &plugins.PluginInstance{
	ID:            uuid.MustParse("c1e81895-1f75-4988-9f10-52786b875ec7"),
	Name:          "task",
	Version:       "1.0.0",
	SkynetVersion: ">= 1.0, < 1.1",
}

type Interface struct{}

// New plugin factory, do NOT change the function name
func NewPlugin() plugins.PluginInterface {
	return &Interface{}
}

var defaultField = log.Fields{
	"plugin": Instance.ID,
}

var (
	taskCancel shared.CancelMap
	pluginAPI  = NewShared()
	sharedKey  = fmt.Sprintf("plugin_%s", Instance.ID.String())
)

func (p *Interface) Instance() *plugins.PluginInstance {
	return Instance
}

func (p *Interface) PluginInit() error {
	utils.GetDB().AutoMigrate(&shared.PluginTask{})
	sn.Skynet.SharedData[sharedKey] = pluginAPI

	sn.Skynet.Page.AddNav([]*sn.SNNavItem{
		{
			Priority: 40,
			Name:     "Task",
			Link:     fmt.Sprintf("/plugin/%s", Instance.ID.String()),
			Icon:     "fa-tasks",
			Role:     sn.RoleUser,
		},
	})
	sn.Skynet.API.AddAPI([]*sn.SNAPIItem{
		{
			Path:   fmt.Sprintf("/plugin/%s/task", Instance.ID.String()),
			Method: sn.APIGet,
			Role:   sn.RoleUser,
			Func:   APIGetAllTask,
		},
		{
			Path:   fmt.Sprintf("/plugin/%s/task", Instance.ID.String()),
			Method: sn.APIDelete,
			Role:   sn.RoleAdmin,
			Func:   APIDeleteInactiveTask,
		},
		{
			Path:   fmt.Sprintf("/plugin/%s/task/:id", Instance.ID.String()),
			Method: sn.APIGet,
			Role:   sn.RoleUser,
			Func:   APIGetTask,
		},
		{
			Path:   fmt.Sprintf("/plugin/%s/task/:id", Instance.ID.String()),
			Method: sn.APIDelete,
			Role:   sn.RoleAdmin,
			Func:   APIKillTask,
		},
	})
	sn.Skynet.Page.AddPage([]*sn.SNPageItem{
		{
			TplName: fmt.Sprintf("plugin_%s_menu", Instance.ID.String()),
			Files:   Instance.WithTplLayerFiles("menu.tmpl"),
			FuncMap: sn.Skynet.Page.GetDefaultFunc(),
			Title:   "Skynet | Task",
			Name:    "Task",
			Link:    fmt.Sprintf("/plugin/%s", Instance.ID.String()),
			Role:    sn.RoleUser,
			Path: sn.Skynet.Page.GetDefaultPath().WithChild([]*sn.SNPathItem{
				{
					Name: "Plugin",
					Link: "/plugin",
				},
				{
					Name:   "Task",
					Active: true,
				},
			}),
			BeforeRender: func(c *gin.Context, u *sn.User, v *sn.SNPageItem) bool {
				count, err := pluginAPI.Count()
				if err != nil {
					log.Error(err)
					c.AbortWithStatus(500)
					return false
				}
				v.Param["_total"] = count
				return true
			},
		},
	})
	return nil
}

func (p *Interface) PluginEnable() error {
	return nil
}

func (p *Interface) PluginDisable() error {
	return nil
}

func (p *Interface) PluginFini() error {
	taskCancel.Range(func(k int, v interface{}) bool {
		v.(func() error)()
		return true
	})
	taskCancel.Clear()
	if viper.GetString("database.type") == "sqlite" {
		utils.GetDB().Model(&shared.PluginTask{}).
			Where("status = ? or status = ?", shared.TaskNotStart, shared.TaskRunning).
			Update("output", gorm.Expr("output || '\nTask force killed by Skynet because of exit.'"))
	} else {
		utils.GetDB().Model(&shared.PluginTask{}).
			Where("status = ? or status = ?", shared.TaskNotStart, shared.TaskRunning).
			Update("output", gorm.Expr("CONCAT(output, '\nTask force killed by Skynet because of exit.')"))
	}
	err := utils.GetDB().Model(&shared.PluginTask{}).
		Where("status = ? or status = ?", shared.TaskNotStart, shared.TaskRunning).
		Update("status", shared.TaskFail).Error
	return err
}

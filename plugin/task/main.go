package main

import (
	plugins "skynet/plugin"
	"skynet/plugin/task/shared"
	"skynet/sn"
	"skynet/sn/utils"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Plugin config, do NOT change the variable name
var Config = plugins.PluginConfig{
	ID:   uuid.MustParse("c1e81895-1f75-4988-9f10-52786b875ec7"), // go https://www.uuidgenerator.net/ to generate your plugin uuid
	Name: "task",                                                 // change to your plugin name
	Dependency: []plugins.PluginDep{
		{
			ID:      uuid.MustParse("2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"),
			Name:    "monitor",
			Version: ">= 1.0, < 1.1",
			Option:  true,
		},
	}, // if your plugin need dependency, write here
	Version:       "1.0.0",         // plugin version, better follow https://semver.org/
	SkynetVersion: ">= 1.0, < 1.1", // skynet version constraints using https://github.com/hashicorp/go-version
	Priority:      0,               // priority to run PluginInit
}

type PluginInstance struct{}

// New plugin factory, do NOT change the function name
func NewPlugin() plugins.PluginInterface {
	return &PluginInstance{}
}

var defaultField = log.Fields{
	"plugin": Config.ID,
}

var taskCancel = make(map[int]func())

var pluginAPI = NewShared()

// PluginInit will be executed after plugin loaded or enabled, return error to stop skynet run or plugin enable
func (p *PluginInstance) PluginInit() error {
	utils.GetDB().AutoMigrate(&shared.PluginTasks{})
	sn.Skynet.SharedData[plugins.SPWithIDPrefix(&Config, "")] = pluginAPI

	sn.Skynet.Page.AddNavItem([]*sn.SNNavItem{
		{
			Priority: 40,
			Name:     "Task",
			Link:     "/plugin/" + Config.ID.String(),
			Icon:     "fa-tasks",
			Role:     sn.RoleUser,
		},
	})
	sn.Skynet.API.AddAPIItem([]*sn.SNAPIItem{
		{
			Path:   plugins.SPWithIDPrefixPath(&Config, "/task"),
			Method: sn.APIGet,
			Role:   sn.RoleUser,
			Func:   APIGetAllTask,
		},
		{
			Path:   plugins.SPWithIDPrefixPath(&Config, "/task"),
			Method: sn.APIDelete,
			Role:   sn.RoleAdmin,
			Func:   APIDeleteInactiveTask,
		},
		{
			Path:   plugins.SPWithIDPrefixPath(&Config, "/task/:id"),
			Method: sn.APIGet,
			Role:   sn.RoleUser,
			Func:   APIGetTask,
		},
		{
			Path:   plugins.SPWithIDPrefixPath(&Config, "/task/:id"),
			Method: sn.APIDelete,
			Role:   sn.RoleAdmin,
			Func:   APIKillTask,
		},
	})
	sn.Skynet.Page.AddPageItem([]*sn.SNPageItem{
		{
			TplName: plugins.SPWithIDPrefix(&Config, "task"),
			Files:   plugins.SPWithLayerFiles("task", "task"),
			FuncMap: sn.Skynet.Page.GetDefaultFunc(),
			Title:   "Skynet | Task",
			Name:    "Task",
			Link:    "/plugin/" + Config.ID.String(),
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
			AfterRenderPrepare: func(c *gin.Context, u *sn.Users, v *sn.SNPageItem) bool {
				count, err := pluginAPI.Count()
				if err != nil {
					log.Error(err)
					c.AbortWithStatus(500)
					return false
				}
				low, high, ok := utils.PreSplitFunc(c, v, int(count), 10, []int{5, 10, 20, 50})
				if !ok {
					return false
				}
				if low == -1 {
					v.Param["tasks"] = []*shared.PluginTasks{}
				} else {
					rec, err := pluginAPI.GetAll([]interface{}{"id desc"}, high-low, low, nil)
					if err != nil {
						log.Error(err)
						c.AbortWithStatus(500)
						return false
					}
					for i := range rec {
						s := strings.Split(rec[i].Output, "\n")
						rec[i].Output = s[len(s)-1]
					}
					v.Param["tasks"] = rec
				}
				return true
			},
		},
	})
	return nil
}

// PluginEnable will be executed when trigger plugin enabled
func (p *PluginInstance) PluginEnable() error {
	return nil
}

// PluginDisable will be executed when trigger plugin disabled, skynet will be reloaded after disabled
func (p *PluginInstance) PluginDisable() error {
	return nil
}

// PluginFini will be executed after plugin disabled or skynet exit
func (p *PluginInstance) PluginFini() error {
	for _, v := range taskCancel {
		v()
	}
	taskCancel = make(map[int]func())
	if viper.GetString("database.type") == "sqlite" {
		utils.GetDB().Model(&shared.PluginTasks{}).
			Where("status = ? or status = ?", shared.TaskNotStart, shared.TaskRunning).
			Update("output", gorm.Expr("output || '\nTask force killed by Skynet because of exit.'"))
	} else {
		utils.GetDB().Model(&shared.PluginTasks{}).
			Where("status = ? or status = ?", shared.TaskNotStart, shared.TaskRunning).
			Update("output", gorm.Expr("CONCAT(output, '\nTask force killed by Skynet because of exit.')"))
	}
	err := utils.GetDB().Model(&shared.PluginTasks{}).
		Where("status = ? or status = ?", shared.TaskNotStart, shared.TaskRunning).
		Update("status", shared.TaskFail).Error
	return err
}

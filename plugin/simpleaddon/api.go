package main

import (
	"context"
	"errors"
	"fmt"
	plugins "skynet/plugin"
	monitor "skynet/plugin/monitor/shared"
	"skynet/plugin/simpleaddon/shared"
	task "skynet/plugin/task/shared"
	"skynet/sn"
	"skynet/sn/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AddonStatus int

const (
	AddonNotInstall AddonStatus = iota
	AddonInstalling
	AddonInstall
	AddonOutdate
	AddonUninstalling
)

var tmpStatus = make(map[uuid.UUID]map[int]AddonStatus)

func TriggerTask(c *plugins.PluginConfig, base int, f shared.TaskFunc) func(ctx context.Context, aid, tid int) error {
	return func(ctx context.Context, aid, tid int) error {
		t := sn.Skynet.SharedData["plugin_c1e81895-1f75-4988-9f10-52786b875ec7"].(task.PluginShared)
		err := t.UpdateStatus(tid, task.TaskRunning)
		if err != nil {
			return err
		}

		err = f(ctx, base, aid, tid)
		if err != nil {
			return err
		}

		delete(tmpStatus[c.ID], aid)
		return nil
	}
}

func CheckInstall(c *plugins.PluginConfig, id int) (bool, error) {
	m := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
	rec, err := m.GetSetting(id, plugins.SPWithIDPrefix(c, "install"))
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && rec.Value != "1") {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func CheckState(c *plugins.PluginConfig, id int) bool {
	m := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
	if _, ok := tmpStatus[c.ID][id]; ok || !m.GetAgents()[id].Online {
		return false
	}
	return true
}

type addonParam struct {
	ID int `uri:"id" binding:"required,min=1"`
}

func install(config *plugins.PluginConfig, id int, installFunc shared.TaskFunc) (int, error) {
	tmpStatus[config.ID][id] = AddonInstalling

	t := sn.Skynet.SharedData["plugin_c1e81895-1f75-4988-9f10-52786b875ec7"].(task.PluginShared)
	err := t.NewCustom(id, "Install Addon", fmt.Sprintf("Install addon\nName: %v  Agent: %v", config.Name, id), func() error {
		delete(tmpStatus[config.ID], id)
		return nil
	}, TriggerTask(config, 100, installFunc))
	if err != nil {
		return 500, err
	}
	return 0, nil
}

func APIInstall(config *plugins.PluginConfig, installFunc shared.TaskFunc) func(c *gin.Context, u *sn.User) (int, error) {
	return func(c *gin.Context, u *sn.User) (int, error) {
		var param addonParam
		err := c.ShouldBindUri(&param)
		if err != nil {
			return 400, err
		}
		fields := log.Fields{
			"ip":     c.ClientIP(),
			"id":     param.ID,
			"plugin": config.ID,
		}

		if !CheckState(config, param.ID) {
			c.JSON(200, gin.H{"code": 1, "msg": "Addon could not be installed now"})
			return 0, nil
		}

		ok, err := CheckInstall(config, param.ID)
		if err != nil {
			return 500, err
		}
		if ok {
			c.JSON(200, gin.H{"code": 1, "msg": "Addon could not be installed now"})
			return 0, nil
		}

		code, err := install(config, param.ID, installFunc)
		if err != nil {
			return code, err
		}
		log.WithFields(fields).Info("Created addon install task")
		c.JSON(200, gin.H{"code": 0, "msg": "Created addon install task"})
		return 0, nil
	}
}

func reinstall(config *plugins.PluginConfig, id int, installFunc shared.TaskFunc, uninstallFunc shared.TaskFunc) (int, error) {
	tmpStatus[config.ID][id] = AddonUninstalling

	t := sn.Skynet.SharedData["plugin_c1e81895-1f75-4988-9f10-52786b875ec7"].(task.PluginShared)
	err := t.NewCustom(id, "Reinstall Addon", fmt.Sprintf("Reinstall addon\nName: %v  Agent: %v", config.Name, id), func() error {
		delete(tmpStatus[config.ID], id)
		return nil
	},
		func(ctx context.Context, agentID, taskID int) error {
			err := TriggerTask(config, 50, uninstallFunc)(ctx, agentID, taskID)
			if err != nil {
				return err
			}
			tmpStatus[config.ID][id] = AddonInstalling
			return TriggerTask(config, 50, installFunc)(ctx, agentID, taskID)
		})
	if err != nil {
		return 500, err
	}
	return 0, nil
}

func APIReinstall(config *plugins.PluginConfig, installFunc shared.TaskFunc, uninstallFunc shared.TaskFunc) func(c *gin.Context, u *sn.User) (int, error) {
	return func(c *gin.Context, u *sn.User) (int, error) {
		var param addonParam
		err := c.ShouldBindUri(&param)
		if err != nil {
			return 400, err
		}
		fields := log.Fields{
			"ip":     c.ClientIP(),
			"id":     param.ID,
			"plugin": config.ID,
		}

		if !CheckState(config, param.ID) {
			c.JSON(200, gin.H{"code": 1, "msg": "Addon could not be reinstalled now"})
			return 0, nil
		}

		ok, err := CheckInstall(config, param.ID)
		if err != nil {
			return 500, err
		}
		if !ok {
			c.JSON(200, gin.H{"code": 1, "msg": "Addon could not be reinstalled now"})
			return 0, nil
		}

		code, err := reinstall(config, param.ID, installFunc, uninstallFunc)
		if err != nil {
			return code, err
		}
		log.WithFields(fields).Info("Created addon reinstall task")
		c.JSON(200, gin.H{"code": 0, "msg": "Created addon reinstall task"})
		return 0, nil
	}
}

func uninstall(config *plugins.PluginConfig, id int, uninstallFunc shared.TaskFunc) (int, error) {
	tmpStatus[config.ID][id] = AddonUninstalling

	t := sn.Skynet.SharedData["plugin_c1e81895-1f75-4988-9f10-52786b875ec7"].(task.PluginShared)
	err := t.NewCustom(id, "Uninstall Addon", fmt.Sprintf("Uninstall addon\nName: %v  Agent: %v", config.Name, id), func() error {
		delete(tmpStatus[config.ID], id)
		return nil
	}, TriggerTask(config, 100, uninstallFunc))
	if err != nil {
		return 500, err
	}
	return 0, nil
}

func APIUninstall(config *plugins.PluginConfig, uninstallFunc shared.TaskFunc) func(c *gin.Context, u *sn.User) (int, error) {
	return func(c *gin.Context, u *sn.User) (int, error) {
		var param addonParam
		err := c.ShouldBindUri(&param)
		if err != nil {
			return 400, err
		}
		fields := log.Fields{
			"ip":     c.ClientIP(),
			"id":     param.ID,
			"plugin": config.ID,
		}

		if !CheckState(config, param.ID) {
			c.JSON(200, gin.H{"code": 1, "msg": "Addon could not be uninstalled now"})
			return 0, nil
		}

		ok, err := CheckInstall(config, param.ID)
		if err != nil {
			return 500, err
		}
		if !ok {
			c.JSON(200, gin.H{"code": 1, "msg": "Addon could not be uninstalled now"})
			return 0, nil
		}

		code, err := uninstall(config, param.ID, uninstallFunc)
		if err != nil {
			return code, err
		}
		log.WithFields(fields).Info("Created addon uninstall task")
		c.JSON(200, gin.H{"code": 0, "msg": "Created addon uninstall task"})
		return 0, nil
	}
}

func APIReinstallAll(config *plugins.PluginConfig, installFunc shared.TaskFunc, uninstallFunc shared.TaskFunc) func(c *gin.Context, u *sn.User) (int, error) {
	return func(c *gin.Context, u *sn.User) (int, error) {
		m := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
		agents := m.GetAgents()
		fields := log.Fields{
			"ip":     c.ClientIP(),
			"plugin": config.ID,
		}
		for _, v := range agents {
			ok, err := CheckInstall(config, v.ID)
			if err != nil {
				return 500, err
			}
			if ok {
				if !CheckState(config, v.ID) {
					continue
				}

				code, err := reinstall(config, v.ID, installFunc, uninstallFunc)
				if err != nil {
					return code, err
				}
			}
		}
		log.WithFields(fields).Info("Created addon all reinstall task")
		c.JSON(200, gin.H{"code": 0, "msg": "Created addon all reinstall task"})
		return 0, nil
	}
}

func APIUninstallAll(config *plugins.PluginConfig, uninstallFunc shared.TaskFunc) func(c *gin.Context, u *sn.User) (int, error) {
	return func(c *gin.Context, u *sn.User) (int, error) {
		m := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
		agents := m.GetAgents()
		fields := log.Fields{
			"ip":     c.ClientIP(),
			"plugin": config.ID,
		}
		for _, v := range agents {
			ok, err := CheckInstall(config, v.ID)
			if err != nil {
				return 500, err
			}
			if ok {
				if !CheckState(config, v.ID) {
					continue
				}

				code, err := uninstall(config, v.ID, uninstallFunc)
				if err != nil {
					return code, err
				}
			}
		}
		log.WithFields(fields).Info("Created addon all uninstall task")
		c.JSON(200, gin.H{"code": 0, "msg": "Created addon all uninstall task"})
		return 0, nil
	}
}

func APIInstallAll(config *plugins.PluginConfig, installFunc shared.TaskFunc) func(c *gin.Context, u *sn.User) (int, error) {
	return func(c *gin.Context, u *sn.User) (int, error) {
		m := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
		agents := m.GetAgents()
		fields := log.Fields{
			"ip":     c.ClientIP(),
			"plugin": config.ID,
		}
		for _, v := range agents {
			ok, err := CheckInstall(config, v.ID)
			if err != nil {
				return 500, err
			}
			if !ok {
				if !CheckState(config, v.ID) {
					continue
				}

				code, err := install(config, v.ID, installFunc)
				if err != nil {
					return code, err
				}
			}
		}
		log.WithFields(fields).Info("Created addon all install task")
		c.JSON(200, gin.H{"code": 0, "msg": "Created addon all install task"})
		return 0, nil
	}
}

func APIGetAddonAgent(config *plugins.PluginConfig) func(c *gin.Context, u *sn.User) (int, error) {
	return func(c *gin.Context, u *sn.User) (int, error) {
		var param plugins.SPPaginationParam
		err := c.ShouldBindQuery(&param)
		if err != nil {
			return 400, err
		}

		type addonInfo struct {
			ID          int
			Name        string
			Online      bool
			Status      AddonStatus
			Version     string
			InstallTime time.Time
		}
		m := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
		agents := m.GetAgents()
		count := len(agents)
		if len(agents) > 0 && (param.Page-1)*param.Size < len(agents) {
			var ret []*addonInfo
			for _, v := range agents {
				ins := addonInfo{
					ID:     v.ID,
					Name:   v.Name,
					Online: v.Online,
				}
				rec, err := m.GetAllSetting(v.ID)
				if err != nil {
					return 500, err
				}
				for _, v := range rec {
					switch v.Name {
					case plugins.SPWithIDPrefix(config, "install"):
						if v.Value == "1" {
							ins.Status = AddonInstall
						}
					case plugins.SPWithIDPrefix(config, "version"):
						ins.Version = v.Value
					case plugins.SPWithIDPrefix(config, "time"):
						t, err := strconv.ParseInt(v.Value, 10, 64)
						if err != nil {
							log.Fatal(err)
						}
						ins.InstallTime = time.Unix(t, 0)
					}
				}
				if v, ok := tmpStatus[config.ID][v.ID]; ok {
					ins.Status = v
				}
				ret = append(ret, &ins)
			}
			c.JSON(200, gin.H{"code": 0, "msg": "Get all addon agent success",
				"data": ret[(param.Page-1)*param.Size : utils.IntMin(param.Page*param.Size, len(ret))], "total": count})
		} else {
			c.JSON(200, gin.H{"code": 0, "msg": "Get all addon agent success", "data": []*addonInfo{}, "total": count})
		}
		return 0, nil
	}
}

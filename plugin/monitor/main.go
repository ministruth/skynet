package main

import (
	"embed"
	"fmt"
	plugins "skynet/plugin"
	"skynet/plugin/monitor/shared"
	"skynet/sn"
	"skynet/sn/tpl"
	"skynet/sn/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

//go:generate sh -c "protoc -I=./proto --go_out=. ./proto/*.proto"

var Instance = &plugins.PluginInfo{
	SNPluginInfo: sn.SNPluginInfo{
		ID:            uuid.MustParse("2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"),
		Name:          "monitor",
		Version:       "1.0.0",
		SkynetVersion: ">= 1.0, < 1.1",
	},
}

type Interface struct{}

// New plugin factory, do NOT change the function name
func NewPlugin() plugins.PluginInterface {
	return &Interface{}
}

var (
	pluginAPI = NewShared()
	//go:embed i18n/*.yml
	i18nFiles     embed.FS
	token         string
	tokenKey      = fmt.Sprintf("plugin_%s_token", Instance.ID.String())
	sharedKey     = fmt.Sprintf("plugin_%s", Instance.ID.String())
	agentInstance tpl.SafeMap[uuid.UUID, *shared.AgentInfo]
)

var configPerm = &sn.PermissionList{
	Name: fmt.Sprintf("plugin.%v.config", Instance.ID),
	Note: "Plugin monitor config",
}

var servicePerm = &sn.PermissionList{
	Name: fmt.Sprintf("plugin.%v.service", Instance.ID),
	Note: "Plugin monitor service",
}

func (p *Interface) Instance() *sn.SNPluginInfo {
	return &Instance.SNPluginInfo
}

func (p *Interface) PluginEnable() error {
	// init setting
	var ok bool
	token, ok = sn.Skynet.Setting.Get(tokenKey)
	if !ok {
		sn.Skynet.Setting.Set(tokenKey, "")
		token = ""
	}
	if token == "" {
		Instance.Log().Warn("Token is empty, generate a token for safety")
	}

	// init agent
	sn.Skynet.GetDB().AutoMigrate(&shared.PluginMonitorAgent{}, &shared.PluginMonitorAgentSetting{})
	rec, err := pluginAPI.GetAllAgent(nil)
	if err != nil {
		return err
	}

	for _, v := range rec {
		agentInstance.Set(v.ID, &shared.AgentInfo{
			ID:        v.ID,
			IP:        v.LastIP,
			Name:      v.Name,
			Hostname:  v.Hostname,
			System:    v.System,
			Machine:   v.Machine,
			LastLogin: v.LastLogin,
			Status:    shared.AgentOffline,
		})
	}

	// init plugin
	if err := Instance.InitPermission(configPerm); err != nil {
		return err
	}

	if err := Instance.InitPermission(servicePerm); err != nil {
		return err
	}

	if err := Instance.ParseLang(i18nFiles); err != nil {
		return err
	}

	Instance.AddPluginMenu(&sn.SNMenu{
		ID:   uuid.MustParse("cd7436da-851b-44ba-80dd-fcf792e8099e"),
		Name: fmt.Sprintf("%v.menu.config", Instance.ID),
		Path: fmt.Sprintf("/plugin/%v/config", Instance.ID),
		Perm: &sn.SNPerm{
			ID:   configPerm.ID,
			Perm: sn.PermRead,
		},
	})

	Instance.AddServiceMenu(&sn.SNMenu{
		ID:   uuid.MustParse("030f1fbb-f9a6-4fdf-9352-73cb62d7dd71"),
		Name: fmt.Sprintf("%v.menu.service", Instance.ID),
		Path: fmt.Sprintf("/plugin/%v/service", Instance.ID),
		Perm: &sn.SNPerm{
			ID:   servicePerm.ID,
			Perm: sn.PermRead,
		},
	})

	sn.Skynet.API.AddAPI([]*sn.SNAPIItem{
		{
			Path:   fmt.Sprintf("/plugin/%v/agent", Instance.ID),
			Method: sn.APIGet,
			Checker: func(perm map[uuid.UUID]*sn.SNPerm) bool {
				if perm != nil {
					if p, ok := perm[configPerm.ID]; ok {
						return (p.Perm & sn.PermRead) == sn.PermRead
					}
					if p, ok := perm[servicePerm.ID]; ok {
						return (p.Perm & sn.PermRead) == sn.PermRead
					}
					if p, ok := perm[sn.Skynet.GetID(sn.PermAllID)]; ok {
						return (p.Perm & sn.PermRead) == sn.PermRead
					}
				}
				return false // fail safe
			},
			Func: APIGetAllAgent,
		},
		{
			Path:   fmt.Sprintf("/plugin/%v/setting", Instance.ID),
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   configPerm.ID,
				Perm: sn.PermRead,
			},
			Func: APIGetSetting,
		},
		{
			Path:   fmt.Sprintf("/plugin/%v/setting", Instance.ID),
			Method: sn.APIPut,
			Perm: &sn.SNPerm{
				ID:   configPerm.ID,
				Perm: sn.PermWriteExecute,
			},
			Func: APIUpdateSetting,
		},
		{
			Path:   fmt.Sprintf("/plugin/%v/ws", Instance.ID),
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermGuestID),
				Perm: sn.PermAll,
			},
			Func: func(c *gin.Context, id uuid.UUID) (int, error) {
				if err := WSHandler(utils.GetIP(c), c.Writer, c.Request); err != nil {
					utils.WithLogTrace(Instance.Log(), err).WithField("ip", utils.GetIP(c)).Error(err)
				}
				return 0, nil
			},
		},
	})

	sn.Skynet.SharedData.Set(sharedKey, pluginAPI)
	return nil
}

func (p *Interface) PluginDisable() error {
	return nil
}

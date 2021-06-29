package main

import (
	plugins "skynet/plugin"
	"skynet/plugin/simpleaddon/shared"
	"skynet/sn"

	"github.com/gin-gonic/gin"
)

func NewShared() shared.PluginShared {
	return &pluginShared{}
}

type pluginShared struct{}

func (s *pluginShared) WithAddonFile(c *plugins.PluginConfig, n string) []string {
	return []string{"templates/home.tmpl", c.Path + "templates/" + n + ".tmpl", Config.Path + "templates/setting.tmpl", "templates/header.tmpl", "templates/footer.tmpl"}
}

func (s *pluginShared) WithAddonParam(base gin.H, c *plugins.PluginConfig, title string, tips string) gin.H {
	if base == nil {
		base = make(gin.H)
	}
	base["_simpleaddon_title"] = title
	base["_simpleaddon_tips"] = tips
	base["_simpleaddon_id"] = c.ID
	return base
}

func (s *pluginShared) WithAddonAPI(c *plugins.PluginConfig, installFunc shared.TaskFunc, uninstallFunc shared.TaskFunc,
	versionFunc func() (string, error)) {
	tmpStatus[c.ID] = make(map[int]AddonStatus)
	sn.Skynet.API.AddAPIItem([]*sn.SNAPIItem{
		{
			Path:   plugins.SPWithIDPrefixPath(c, "/agent"),
			Method: sn.APIGet,
			Role:   sn.RoleAdmin,
			Func:   APIGetAddonAgent(c),
		},
		{
			Path:   plugins.SPWithIDPrefixPath(c, "/agent"),
			Method: sn.APIPost,
			Role:   sn.RoleAdmin,
			Func:   APIInstallAll(c, installFunc),
		},
		{
			Path:   plugins.SPWithIDPrefixPath(c, "/agent"),
			Method: sn.APIPatch,
			Role:   sn.RoleAdmin,
			Func:   APIReinstallAll(c, installFunc, uninstallFunc),
		},
		{
			Path:   plugins.SPWithIDPrefixPath(c, "/agent"),
			Method: sn.APIDelete,
			Role:   sn.RoleAdmin,
			Func:   APIUninstallAll(c, uninstallFunc),
		},
		{
			Path:   plugins.SPWithIDPrefixPath(c, "/version"),
			Method: sn.APIGet,
			Role:   sn.RoleAdmin,
			Func: func(c *gin.Context, u *sn.User) (int, error) {
				ver, err := versionFunc()
				if err != nil {
					return 500, err
				}
				c.JSON(200, gin.H{"code": 0, "msg": "Get version success", "data": ver})
				return 0, nil
			},
		},
		{
			Path:   plugins.SPWithIDPrefixPath(c, "/agent/:id"),
			Method: sn.APIPost,
			Role:   sn.RoleAdmin,
			Func:   APIInstall(c, installFunc),
		},
		{
			Path:   plugins.SPWithIDPrefixPath(c, "/agent/:id"),
			Method: sn.APIDelete,
			Role:   sn.RoleAdmin,
			Func:   APIUninstall(c, uninstallFunc),
		},
		{
			Path:   plugins.SPWithIDPrefixPath(c, "/agent/:id"),
			Method: sn.APIPatch,
			Role:   sn.RoleAdmin,
			Func:   APIReinstall(c, installFunc, uninstallFunc),
		},
	})
}

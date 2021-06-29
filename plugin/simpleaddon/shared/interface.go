package shared

import (
	"context"
	"errors"
	plugins "skynet/plugin"

	"github.com/gin-gonic/gin"
)

var (
	PlatformNotSupportError = errors.New("Platform not supported")
)

type TaskFunc func(ctx context.Context, base, aid, tid int) error

type PluginShared interface {
	WithAddonFile(c *plugins.PluginConfig, n string) []string
	WithAddonParam(base gin.H, c *plugins.PluginConfig, title string, tips string) gin.H
	WithAddonAPI(c *plugins.PluginConfig, installFunc TaskFunc, uninstallFunc TaskFunc,
		versionFunc func() (string, error))
}

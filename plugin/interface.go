package plugins

import (
	"skynet/sn"

	"github.com/google/uuid"
)

type PluginDep struct {
	ID      uuid.UUID
	Name    string
	Version string
	Option  bool
}

type PluginConfig struct {
	ID            uuid.UUID
	Name          string
	Dependency    []PluginDep
	Path          string
	Version       string
	SkynetVersion string
	Priority      int
}

type PluginInterface interface {
	PluginInit() error
	PluginEnable() error
	PluginDisable() error
	PluginFini() error
}

type SPPaginationParam struct {
	Order string `form:"order,default=asc" binding:"oneof=asc desc"`
	Page  int    `form:"page,default=1" binding:"min=1"`
	Size  int    `form:"size,default=10"`
}

func SPWithIDPrefixPath(c *PluginConfig, p string) string {
	return "/plugin/" + c.ID.String() + p
}

func SPWithIDPrefix(c *PluginConfig, n string) string {
	if n != "" {
		return "plugin_" + c.ID.String() + "_" + n
	} else {
		return "plugin_" + c.ID.String()
	}
}

func SPAddSubPath(root string, i []*sn.SNNavItem) {
	for _, v := range sn.Skynet.Page.GetNavItem() {
		if v.Name == root {
			v.Child = append(v.Child, i...)
			sn.SNNavSort(v.Child).Sort()
		}
	}
}

func SPWithLayerFiles(pn string, n string) []string {
	return []string{"templates/home.tmpl", "plugin/" + pn + "/templates/" + n + ".tmpl", "templates/header.tmpl", "templates/footer.tmpl"}
}

func SPWithSingleFiles(pn string, n string) []string {
	return []string{"plugin/" + pn + "/templates/" + n + ".tmpl", "templates/header.tmpl", "templates/footer.tmpl"}
}

package plugins

import (
	"skynet/sn"

	"github.com/google/uuid"
)

type PluginDep struct {
	ID      uuid.UUID
	Name    string
	Version string
}

type PluginConfig struct {
	ID            uuid.UUID
	Name          string
	Dependency    []PluginDep
	Path          string
	Version       string
	SkynetVersion string
}

func SPWithIDPrefix(c *PluginConfig, n string) string {
	if n != "" {
		return "plugin_" + c.ID.String() + "_" + n
	} else {
		return "plugin_" + c.ID.String()
	}
}

func SPAddSubPath(root string, name string, link string, icon string, role sn.UserRole, front bool) {
	ins := sn.SNPageItem{
		Name: name,
		Link: link,
		Icon: icon,
		Role: role,
	}
	for _, v := range sn.Skynet.Page.GetPage() {
		if v.Name == root {
			if front {
				v.Child = append([]*sn.SNPageItem{&ins}, v.Child...)
			} else {
				v.Child = append(v.Child, &ins)
			}
		}
	}
}

func SPAddTemplate(p string, n string, f string) {
	sn.Skynet.Page.AddTemplate(n, "templates/home.tmpl", "plugin/"+p+"/"+f, "templates/header.tmpl", "templates/footer.tmpl")
}

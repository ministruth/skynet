package plugin

import (
	"path"
	"skynet/sn"

	"github.com/google/uuid"
)

// PluginInstance is plugin instance struct.
type PluginInstance struct {
	ID            uuid.UUID // plugin unique ID
	Name          string    // plugin name, unique suggested
	Version       string    // plugin version
	Path          string    // auto filled, runtime absolute path, no ending / unless root
	SkynetVersion string    // compatible skynet version
}

// PluginInterface is plugin interface, every plugin should export one with NewPlugin function.
// Signature: func NewPlugin() PluginInterface
type PluginInterface interface {
	// Instance will return the instance of the plugin, skynet will fill the runtime fields of it.
	// You should make sure the instance returned will be persist and the same each time calls.
	Instance() *PluginInstance

	// PluginInit will be executed when plugin loaded or enabled, return error to stop plugin enable,
	// note that the plugin initialize order may not stable.
	PluginInit() error

	// PluginEnable will be executed when plugin enabled, before PluginInit,
	// return error to stop plugin enable.
	PluginEnable() error

	// PluginDisable will be executed when plugin disabled, after PluginFini.
	// return error to stop plugin disable.
	// Note that skynet will be reloaded after disabled.
	PluginDisable() error

	// PluginFini will be executed when plugin disabled or skynet exit,
	// return error to stop plugin disable.
	PluginFini() error
}

// PaginationParam is the common pagination param for plugins.
type PaginationParam struct {
	Order string `form:"order,default=asc" binding:"oneof=asc desc"`
	Page  int    `form:"page,default=1" binding:"min=1"`
	Size  int    `form:"size,default=10"`
}

// GetTempFilePath returns the relative temp file path with suffix.
func (p *PluginInstance) GetTempFilePath(suffix string) string {
	return path.Join("temp/plugin", p.ID.String(), suffix)
}

// GetDataFilePath returns the relative data file path with suffix.
func (p *PluginInstance) GetDataFilePath(suffix string) string {
	return path.Join("data/plugin", p.ID.String(), suffix)
}

// AddSubNav will add sub item item to navbar ID root.
func (p *PluginInstance) AddSubNav(root string, item []*sn.SNNavItem) {
	for _, v := range sn.Skynet.Page.GetNav() {
		if v.ID == root {
			v.Child = append(v.Child, item...)
			sn.SNNavSort(v.Child).Sort()
			break
		}
	}
}

// AddStaticRouter will add relativePath to static file routerPath.
func (p *PluginInstance) AddStaticRouter(routerPath string, relativePath string) {
	sn.Skynet.StaticFile.Static(routerPath, path.Join(p.Path, relativePath))
}

// WithTplLayerFiles return template file list including panel file and tplFile.
// Note that template file must be stored in templates sub folder of plugin.
func (p *PluginInstance) WithTplLayerFiles(tplFile string) []string {
	return []string{"templates/home.tmpl", path.Join(p.Path, "templates", tplFile), "templates/header.tmpl", "templates/footer.tmpl"}
}

// WithSingleFiles return template file list including common file and tplFile.
// Note that template file must be stored in templates sub folder of plugin.
func (p *PluginInstance) WithSingleFiles(tplFile string) []string {
	return []string{path.Join(p.Path, "templates", tplFile), "templates/header.tmpl", "templates/footer.tmpl"}
}

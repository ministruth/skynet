package handler

import (
	"errors"
	"io/ioutil"
	"plugin"
	plugins "skynet/plugin"
	"skynet/sn"
	"skynet/sn/utils"
	"sort"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type PluginLoad struct {
	*plugins.PluginConfig
	Enable        bool
	DisableReason string
	Instance      plugins.PluginInterface `json:"-"`
	Loader        *plugin.Plugin          `json:"-"`
}

var (
	PluginNotFoundError = errors.New("Plugin not found")
)

type pluginLoadSort []*PluginLoad

func (s pluginLoadSort) Len() int           { return len(s) }
func (s pluginLoadSort) Less(i, j int) bool { return s[i].Priority < s[j].Priority }
func (s pluginLoadSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type sitePlugin struct {
	priority pluginLoadSort
	plugin   map[uuid.UUID]*PluginLoad
}

func (p *sitePlugin) readPlugin(base string) error {
	files, err := ioutil.ReadDir(base)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			dirFile, err := ioutil.ReadDir(base + "/" + f.Name())
			if err != nil {
				return err
			}
			for _, df := range dirFile {
				if strings.HasSuffix(df.Name(), ".so") {
					soFile := base + "/" + f.Name() + "/" + df.Name()
					pPlugin, err := plugin.Open(soFile)
					if err != nil {
						log.Error("Can't open plugin file: ", soFile)
						return err
					}
					pSymbol, err := pPlugin.Lookup("Config")
					if err != nil {
						log.Error("Can't locate plugin config: ", soFile)
						return err
					}
					pConfig := *pSymbol.(**plugins.PluginConfig)
					pConfig.Path = base + "/" + f.Name() + "/"
					pSymbol, err = pPlugin.Lookup("NewPlugin")
					if err != nil {
						log.Error("Can't locate plugin entry: ", soFile)
						return err
					}
					pInstance := pSymbol.(func() plugins.PluginInterface)()
					p.plugin[pConfig.ID] = &PluginLoad{
						PluginConfig: pConfig,
						Enable:       false,
						Instance:     pInstance,
						Loader:       pPlugin,
					}
					p.priority = append(p.priority, p.plugin[pConfig.ID])
					log.WithFields(log.Fields{
						"id":      pConfig.ID,
						"name":    pConfig.Name,
						"version": pConfig.Version,
					}).Info("Plugin loaded")
				}
			}
		}
	}

	sort.Stable(p.priority)
	return nil
}

func (p *sitePlugin) cleanPlugin() error {
	setting := sn.Skynet.Setting.GetCache()
	// setting enable cleanup
	for k, v := range setting {
		if strings.HasPrefix(k, "plugin_") && v == "1" {
			setting[k] = "-1"
		}
	}

	for _, v := range p.priority {
		if status, exist := setting["plugin_"+v.ID.String()]; exist {
			v.Enable = status == "-1"
			if v.Enable {
				setting["plugin_"+v.ID.String()] = "1"
			}
		} else {
			err := sn.Skynet.Setting.New("plugin_"+v.ID.String(), "0")
			if err != nil {
				return err
			}
		}
	}

	for k, v := range setting {
		if strings.HasPrefix(k, "plugin_") && v == "-1" {
			err := sn.Skynet.Setting.Delete(k)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *sitePlugin) checkVersion(showLog bool) error {
	for _, v := range p.priority {
		c, err := utils.CheckSkynetVersion(v.SkynetVersion)
		if err != nil {
			return err
		}
		if !c {
			v.Enable = false
			v.DisableReason = "Skynet version mismatch, need " + v.SkynetVersion
			sn.Skynet.Setting.Update("plugin_"+v.ID.String(), "0")
			if showLog {
				log.Errorf("Plugin %v need skynet version %v, disable now.", v.Name, v.SkynetVersion)
			}
		}
	}
	return nil
}

func (p *sitePlugin) checkDependency(showLog bool) error {
	// dependency check
	for _, v := range p.priority {
		if v.DisableReason == "" {
			for _, dep := range v.Dependency {
				if depPlugin, exist := p.plugin[dep.ID]; exist {
					if depPlugin.Enable == false {
						if dep.Option {
							if v.Enable == true && showLog {
								log.Warnf("Plugin %v recommand enable dependency %v version %v to get full experience.", v.Name, dep.Name, dep.Version)
							}
						} else {
							v.Enable = false
							v.DisableReason = "Need to enable dependency " + dep.Name + "(" + dep.ID.String() + ")"
							sn.Skynet.Setting.Update("plugin_"+v.ID.String(), "0")
							if showLog {
								log.Warnf("Plugin %v need dependency %v to be enabled, disable now.", v.Name, dep.Name)
							}
							break
						}
					}
					c, err := utils.CheckVersion(depPlugin.Version, dep.Version)
					if err != nil {
						return err
					}
					if !c {
						v.Enable = false
						v.DisableReason = "Need dependency " + dep.Name + " version " + dep.Version
						sn.Skynet.Setting.Update("plugin_"+v.ID.String(), "0")
						if showLog {
							log.Warnf("Plugin %v need dependency %v version %v, disable now.", v.Name, dep.Name, dep.Version)
						}
						break
					}
				} else {
					if dep.Option {
						if v.Enable == true && showLog {
							log.Warnf("Plugin %v recommand install dependency %v version %v to get full experience.", v.Name, dep.Name, dep.Version)
						}
					} else {
						v.Enable = false
						v.DisableReason = "Need install dependency " + dep.Name + "(" + dep.ID.String() + ")"
						sn.Skynet.Setting.Update("plugin_"+v.ID.String(), "0")
						if showLog {
							log.Errorf("Plugin %v need dependency %v(%v), disable now.", v.Name, dep.Name, dep.ID.String())
						}
						break
					}
				}
			}
		}
	}
	return nil
}

func NewPlugin(base string) (sn.SNPlugin, error) {
	var ret sitePlugin
	ret.plugin = make(map[uuid.UUID]*PluginLoad)

	err := ret.readPlugin(base)
	if err != nil {
		return nil, err
	}

	err = ret.cleanPlugin()
	if err != nil {
		return nil, err
	}

	err = ret.checkVersion(true)
	if err != nil {
		return nil, err
	}

	err = ret.checkDependency(true)
	if err != nil {
		return nil, err
	}

	// plugin init
	for _, v := range ret.priority {
		if v.Enable {
			err = v.Instance.PluginInit()
			if err != nil {
				return nil, err
			}
		}
	}

	return &ret, nil
}

func (p *sitePlugin) GetAll() interface{} {
	return p.plugin
}

func (p *sitePlugin) Get(id uuid.UUID) interface{} {
	v, exist := p.plugin[id]
	if !exist {
		return nil
	}
	return v
}

func (p *sitePlugin) Disable(id uuid.UUID) error {
	if v, exist := p.plugin[id]; exist {
		if !v.Enable {
			return nil
		}
		err := v.Instance.PluginDisable()
		if err != nil {
			return err
		}
		err = v.Instance.PluginFini()
		if err != nil {
			return err
		}
		v.Enable = false
		sn.Skynet.Setting.Update("plugin_"+v.ID.String(), "0")
		return nil
	}
	return PluginNotFoundError
}

func (p *sitePlugin) Enable(id uuid.UUID) error {
	if v, exist := p.plugin[id]; exist {
		if v.Enable {
			return nil
		}
		if v.DisableReason != "" {
			return errors.New(v.DisableReason)
		}
		for _, dp := range v.Dependency {
			err := p.Enable(dp.ID)
			if err != nil {
				return err
			}
		}
		err := v.Instance.PluginEnable()
		if err != nil {
			return err
		}
		err = v.Instance.PluginInit()
		if err != nil {
			return err
		}
		v.Enable = true
		sn.Skynet.Setting.Update("plugin_"+v.ID.String(), "1")
		for _, v := range p.priority {
			v.DisableReason = ""
		}
		err = p.checkVersion(false)
		if err != nil {
			return err
		}
		err = p.checkDependency(false)
		if err != nil {
			return err
		}
		return nil
	}
	return PluginNotFoundError
}

func (p *sitePlugin) Fini() {
	for i := len(p.priority) - 1; i >= 0; i-- {
		if p.priority[i].Enable {
			err := p.priority[i].Instance.PluginFini()
			if err != nil {
				log.Warnf("Plugin %v fini error: %v", p.priority[i].Name, err)
			}
		}
	}
}

func (p *sitePlugin) Count() int64 {
	return int64(len(p.plugin))
}

package handler

import (
	"errors"
	"io/ioutil"
	"plugin"
	plugins "skynet/plugin"
	"skynet/sn"
	"skynet/sn/utils"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type pluginLoad struct {
	plugins.PluginConfig
	Enable        bool
	DisableReason string
	Instance      *plugin.Plugin
}

func (p *pluginLoad) CallPlugin(n string) error {
	pSymbol, err := p.Instance.Lookup(n)
	if err == nil {
		return pSymbol.(func() error)()
	}
	return nil
}

type sitePlugin struct {
	plugin map[uuid.UUID]*pluginLoad
}

func NewPlugin(base string) (sn.SNPlugin, error) {
	var ret sitePlugin
	ret.plugin = make(map[uuid.UUID]*pluginLoad)

	files, err := ioutil.ReadDir(base)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.IsDir() {
			pfiles, err := ioutil.ReadDir(base + "/" + f.Name())
			if err != nil {
				return nil, err
			}
			for _, pf := range pfiles {
				if strings.HasSuffix(pf.Name(), ".so") {
					p, err := plugin.Open(base + "/" + f.Name() + "/" + pf.Name())
					if err != nil {
						log.Error("Error loading plugin: ", base+"/"+f.Name()+"/"+pf.Name())
						return nil, err
					}
					pSymbol, err := p.Lookup("Config")
					if err != nil {
						log.Error("Error loading plugin: ", base+"/"+f.Name()+"/"+pf.Name())
						return nil, err
					}
					pConfig := *pSymbol.(*plugins.PluginConfig)
					ret.plugin[pConfig.ID] = &pluginLoad{
						PluginConfig: pConfig,
						Enable:       false,
						Instance:     p,
					}
					log.WithFields(log.Fields{
						"id":      pConfig.ID,
						"name":    pConfig.Name,
						"version": pConfig.Version,
					}).Info("Plugin loaded")
				}
			}
		}
	}

	// setting enable cleanup
	for k, v := range sn.Skynet.Setting.Get() {
		if strings.HasPrefix(k, "plugin_") && v == "1" {
			sn.Skynet.Setting.Get()[k] = "-1"
		}
	}

	for k, v := range ret.plugin {
		if status, exist := sn.Skynet.Setting.GetSetting("plugin_" + k.String()); exist {
			v.Enable = status == "-1"
			if v.Enable {
				sn.Skynet.Setting.Get()["plugin_"+k.String()] = "1"
			}
		} else {
			err := sn.Skynet.Setting.AddSetting("plugin_"+k.String(), "0")
			if err != nil {
				return nil, err
			}
		}
	}

	for k, v := range sn.Skynet.Setting.Get() {
		if strings.HasPrefix(k, "plugin_") && v == "-1" {
			err := sn.Skynet.Setting.DelSetting(k)
			if err != nil {
				return nil, err
			}
		}
	}

	// skynet version check
	for _, v := range ret.plugin {
		c, err := utils.CheckSkynetVersion(v.SkynetVersion)
		if err != nil {
			return nil, err
		}
		if !c {
			v.Enable = false
			v.DisableReason = "Skynet version mismatch, need " + v.SkynetVersion
			sn.Skynet.Setting.EditSetting("plugin_"+v.ID.String(), "0")
			log.Errorf("Plugin %v need skynet version %v, disable now.", v.Name, v.SkynetVersion)
		}
	}

	// dependency check
	for _, v := range ret.plugin {
		for _, p := range v.Dependency {
			if dp, exist := ret.plugin[p.ID]; exist {
				if dp.Enable == false && v.Enable == true {
					v.Enable = false
					v.DisableReason = "Need to enable dependency " + p.Name + "(" + p.ID.String() + ")"
					sn.Skynet.Setting.EditSetting("plugin_"+v.ID.String(), "0")
					log.Warnf("Plugin %v need dependency %v to be enabled, disable now.", v.Name, p.Name)
				}
				c, err := utils.CheckVersion(dp.Version, p.Version)
				if err != nil {
					return nil, err
				}
				if !c {
					v.Enable = false
					v.DisableReason = "Need dependency " + p.Name + " version " + p.Version
					sn.Skynet.Setting.EditSetting("plugin_"+v.ID.String(), "0")
					log.Warnf("Plugin %v need dependency %v version %v, disable now.", v.Name, p.Name, p.Version)
				}
			} else {
				v.Enable = false
				v.DisableReason = "Need install dependency " + p.Name + "(" + p.ID.String() + ")"
				sn.Skynet.Setting.EditSetting("plugin_"+v.ID.String(), "0")
				log.Errorf("Plugin %v need dependency %v(%v), disable now.", v.Name, p.Name, p.ID.String())
			}
		}
	}

	// plugin init
	for _, v := range ret.plugin {
		if v.Enable {
			err = v.CallPlugin("PluginInit")
			if err != nil {
				return nil, err
			}
		}
	}

	return &ret, nil
}

func (p *sitePlugin) GetAllPlugin() interface{} {
	return p.plugin
}

func (p *sitePlugin) GetPlugin(id uuid.UUID) interface{} {
	v, exist := p.plugin[id]
	if !exist {
		return nil
	}
	return v
}

func (p *sitePlugin) DisablePlugin(id uuid.UUID) error {
	if v, exist := p.plugin[id]; exist {
		err := v.CallPlugin("PluginDisable")
		if err != nil {
			return err
		}
		v.Enable = false
		sn.Skynet.Setting.EditSetting("plugin_"+v.ID.String(), "0")
		return nil
	}
	return errors.New("Plugin not found")
}

func (p *sitePlugin) EnablePlugin(id uuid.UUID) error {
	if v, exist := p.plugin[id]; exist {
		if v.DisableReason != "" {
			return errors.New(v.DisableReason)
		}
		err := v.CallPlugin("PluginEnable")
		if err != nil {
			return err
		}
		err = v.CallPlugin("PluginInit")
		if err != nil {
			return err
		}
		v.Enable = true
		sn.Skynet.Setting.EditSetting("plugin_"+v.ID.String(), "1")
		return nil
	}
	return errors.New("Plugin not found")
}

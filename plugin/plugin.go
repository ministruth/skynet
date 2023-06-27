package plugin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"gopkg.in/yaml.v2"
)

const pluginSettingPrefix = "plugin_"
const pluginConfig = "config.yml"

type PluginImpl struct {
	entry map[uuid.UUID]*sn.PluginEntry
}

func NewPlugin(dir string) sn.Plugin {
	ret := &PluginImpl{entry: make(map[uuid.UUID]*sn.PluginEntry)}
	ret.parseFolder(dir)
	ret.checkSetting()
	ret.loadPlugin()
	// plugin enable
	for _, v := range ret.entry {
		if v.Enable {
			if err := v.Instance.PluginEnable(); err != nil {
				log.NewEntry(err).WithFields(log.F{
					"id":   v.ID,
					"name": v.Name,
					"path": v.Path,
				}).Error("Enable plugin error")
				v.Enable = false
				ret.updateSetting(v.ID, false)
			} else {
				log.New().WithFields(log.F{
					"id":   v.ID,
					"name": v.Name,
					"path": v.Path,
				}).Debugf("Plugin %v enabled", v.Name)
			}
		} else {
			log.New().WithFields(log.F{
				"id":   v.ID,
				"name": v.Name,
				"path": v.Path,
			}).Debugf("Plugin %v disabled", v.Name)
		}
	}
	return ret
}

func (impl *PluginImpl) updateSetting(id uuid.UUID, enable bool) {
	var val string
	if enable {
		val = "1"
	} else {
		val = "0"
	}
	if err := sn.Skynet.Setting.Set(pluginSettingPrefix+id.String(), val); err != nil {
		log.NewEntry(err).Error("Cannot set plugin setting")
	}
}

func (impl *PluginImpl) Delete(id uuid.UUID) error {
	entry := impl.Get(id)
	if entry == nil {
		return sn.ErrPluginNotFound
	}
	if entry.Instance != nil {
		return sn.ErrPluginLoaded
	}
	if err := tracerr.Wrap(os.RemoveAll(entry.Path)); err != nil {
		return err
	}
	delete(impl.entry, id)
	return nil
}

func (impl *PluginImpl) Enable(id uuid.UUID) error {
	entry := impl.Get(id)
	if entry == nil {
		return sn.ErrPluginNotFound
	}
	if entry.Enable {
		return nil
	}
	if entry.Instance == nil {
		if err := impl.Load(id); err != nil {
			return err
		}
	}
	if err := entry.Instance.PluginEnable(); err != nil {
		return err
	}
	impl.updateSetting(id, true)
	entry.Enable = true
	return nil
}

func (impl *PluginImpl) Disable(id uuid.UUID) error {
	entry := impl.Get(id)
	if entry == nil {
		return sn.ErrPluginNotFound
	}
	if !entry.Enable {
		return nil
	}
	if entry.Instance != nil {
		if err := entry.Instance.PluginDisable(); err != nil {
			return err
		}
	}
	impl.updateSetting(id, false)
	entry.Enable = false
	return nil
}

func (impl *PluginImpl) Get(id uuid.UUID) *sn.PluginEntry {
	ret, ok := impl.entry[id]
	if ok {
		return ret
	}
	return nil
}

func (impl *PluginImpl) loadPlugin() {
	for k, v := range impl.entry {
		if v.Enable && v.Loader == nil && v.Instance == nil {
			if err := impl.Load(k); err != nil {
				log.NewEntry(err).WithFields(log.F{
					"id":   v.ID,
					"name": v.Name,
					"path": v.Path,
				}).Error("Load plugin error")
				impl.updateSetting(k, false)
				v.Enable = false
			}
		}
	}
}

func (impl *PluginImpl) parseFolder(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.NewEntry(tracerr.Wrap(err)).Fatal(err)
	}
	for _, f := range files {
		if f.IsDir() {
			path := filepath.Join(dir, f.Name())
			if _, err := impl.Parse(path); err != nil {
				log.NewEntry(err).WithField("dir", path).Error("Parse plugin error")
			}
		}
	}
}

func (impl *PluginImpl) Unload() {
	for _, v := range impl.entry {
		if v.Instance != nil {
			if err := v.Instance.PluginUnload(); err != nil {
				log.NewEntry(err).WithFields(log.F{
					"id":   v.ID,
					"name": v.Name,
					"path": v.Path,
				}).Error("Unload plugin error")
			}
		}
	}
}

func (impl *PluginImpl) GetAll() []*sn.PluginEntry {
	return utils.MapValueToSlice(impl.entry)
}

func (impl *PluginImpl) Parse(dir string) (*sn.PluginEntry, error) {
	if !utils.FileExist(filepath.Join(dir, pluginConfig)) {
		return nil, nil
	}
	var sofile string
	dirfile, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	for _, df := range dirfile {
		if strings.HasSuffix(df.Name(), ".so") {
			sofile = filepath.Join(dir, df.Name())
			break
		}
	}
	if sofile == "" {
		return nil, nil
	}

	config, err := os.ReadFile(filepath.Join(dir, pluginConfig))
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	entry := new(sn.PluginEntry)
	if err = tracerr.Wrap(yaml.Unmarshal(config, entry)); err != nil {
		return nil, err
	}
	if v := impl.Get(entry.ID); v != nil {
		return nil, tracerr.Wrap(fmt.Errorf("%w: %v and %v have same ID %v", sn.ErrPluginIDDuplicate, entry.Name, v.Name, v.ID))
	}

	entry.LibPath = sofile
	entry.Path = dir
	impl.entry[entry.ID] = entry
	log.New().WithFields(log.F{
		"id":   entry.ID,
		"name": entry.Name,
		"path": entry.Path,
	}).Debugf("Plugin %v parsed", entry.Name)
	return entry, nil
}

func (impl *PluginImpl) Load(id uuid.UUID) error {
	entry := impl.Get(id)
	if entry == nil {
		return sn.ErrPluginNotFound
	}

	p, err := plugin.Open(entry.LibPath)
	if err != nil {
		return tracerr.Wrap(err)
	}
	sym, err := p.Lookup("NewPlugin")
	if err != nil {
		return tracerr.Wrap(err)
	}
	instfunc, ok := sym.(func() sn.PluginInstance)
	if !ok {
		return sn.ErrPluginInvalid
	}
	entry.Instance = instfunc()
	if err := entry.Instance.PluginLoad(); err != nil {
		return err
	}
	entry.Loader = p

	log.New().WithFields(log.F{
		"id":   entry.ID,
		"name": entry.Name,
		"path": entry.Path,
	}).Debugf("Plugin %v loaded", entry.Name)
	return nil
}

func (impl *PluginImpl) checkSetting() {
	setting := sn.Skynet.Setting.GetAll()
	valid := make(map[string]bool)

	for k, v := range impl.entry {
		name := pluginSettingPrefix + k.String()
		if status, exist := setting[name]; exist {
			v.Enable = status == "1"
			valid[name] = true
		} else {
			impl.updateSetting(k, false)
		}
	}

	// setting cleanup
	for k := range setting {
		if strings.HasPrefix(k, pluginSettingPrefix) {
			if !utils.MapContains(valid, k) {
				if _, err := sn.Skynet.Setting.Delete(k); err != nil {
					log.NewEntry(err).Error("Cannot delete plugin setting")
				}
			}
		}
	}
}

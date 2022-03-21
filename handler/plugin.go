package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"plugin"
	plugins "skynet/plugin"
	"skynet/sn"
	"skynet/sn/tpl"
	"skynet/sn/utils"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

const settingPrefix = "plugin_"

var (
	ErrPluginNotFound    = tracerr.New("plugin not found")
	ErrPluginIDDuplicate = tracerr.New("plugin ID duplicated")
	ErrPluginInvalid     = tracerr.New("plugin invalid")
	ErrPluginExists      = tracerr.New("plugin already exists")
	ErrPluginIDNotMatch  = tracerr.New("plugin id not match")
)

type sitePlugin struct {
	plugin     *tpl.SafeMap[uuid.UUID, *sn.SNPluginEntry]
	baseFolder string
}

func (p *sitePlugin) loadPlugin(path string) error {
	pPlugin, err := plugin.Open(path)
	if err != nil {
		return tracerr.Wrap(err)
	}
	pSymbol, err := pPlugin.Lookup("NewPlugin")
	if err != nil {
		return tracerr.Wrap(err)
	}
	pInterface := pSymbol.(func() plugins.PluginInterface)()
	pInstance := pInterface.Instance()
	if v, ok := p.plugin.Get(pInstance.ID); ok {
		return fmt.Errorf("%w: %v and %v have same ID %v", ErrPluginIDDuplicate, v.Name, pInstance.Name, v.ID)
	}
	p.plugin.Set(pInstance.ID, &sn.SNPluginEntry{
		SNPluginInfo: pInstance,
		Path:         filepath.Dir(path),
		Enable:       false,
		Interface:    pInterface,
		Loader:       pPlugin,
	})
	log.WithFields(log.Fields{
		"id":      pInstance.ID,
		"name":    pInstance.Name,
		"version": pInstance.Version,
	}).Info("Plugin loaded")
	return nil
}

func (p *sitePlugin) loadPluginFolder(dir string) error {
	dirFile, err := ioutil.ReadDir(dir)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, df := range dirFile {
		if strings.HasSuffix(df.Name(), ".so") {
			soFile := dir + "/" + df.Name()
			if err := tracerr.Wrap(p.loadPlugin(soFile)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *sitePlugin) cleanSetting() {
	setting := sn.Skynet.Setting.GetAll()
	// setting enable cleanup
	for k, v := range setting {
		if strings.HasPrefix(k, settingPrefix) && v == "1" {
			setting[k] = "-1"
		}
	}

	p.plugin.Range(func(k uuid.UUID, v *sn.SNPluginEntry) bool {
		name := settingPrefix + v.ID.String()
		if status, ok := setting[name]; ok {
			v.Enable = status == "-1"
			if v.Enable {
				setting[name] = "1"
			}
		} else {
			if err := sn.Skynet.Setting.Set(name, "0"); err != nil {
				utils.WithTrace(err).Error(err)
			}
		}
		return true
	})

	for k, v := range setting {
		if strings.HasPrefix(k, settingPrefix) && v == "-1" {
			if _, err := sn.Skynet.Setting.Delete(k); err != nil {
				utils.WithTrace(err).Error(err)
			}
		}
	}
}

func (p *sitePlugin) checkPlugin(v *sn.SNPluginEntry) bool {
	// check version
	c, err := utils.CheckVersion(sn.VERSION, v.SkynetVersion)
	if err != nil {
		utils.WithTrace(err).Errorf("%v: Version constraint %v invalid (%v)", ErrPluginInvalid, v.SkynetVersion, err.Error())
	}
	if !c {
		v.Disable(fmt.Sprintf("Skynet version mismatch, need %s", v.SkynetVersion))
		log.Errorf("Plugin %v need skynet version %v, disable now.", v.Name, v.SkynetVersion)
		if err := sn.Skynet.Setting.Set(settingPrefix+v.ID.String(), "0"); err != nil {
			utils.WithTrace(err).Error(err)
		}
		return false
	}

	return true
}

func (p *sitePlugin) Call(cb sn.SNPluginCBType, param interface{}) []error {
	var ret []error = nil
	p.plugin.Range(func(k uuid.UUID, v *sn.SNPluginEntry) bool {
		if v.Enable && v.Callback != nil {
			if f, ok := v.Callback[cb]; ok && f != nil {
				if err := f(param); err != nil {
					ret = append(ret, err)
				}
			}
		}
		return true
	})
	return ret
}

func NewPlugin(base string) (sn.SNPlugin, error) {
	var ret sitePlugin
	ret.baseFolder = base
	ret.plugin = new(tpl.SafeMap[uuid.UUID, *sn.SNPluginEntry])

	files, err := ioutil.ReadDir(base)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	for _, f := range files {
		if f.IsDir() {
			if err := ret.loadPluginFolder(path.Join(base, f.Name())); err != nil {
				utils.WithTrace(err).Error(err)
			}
		}
	}
	ret.cleanSetting()
	ret.plugin.Range(func(k uuid.UUID, v *sn.SNPluginEntry) bool {
		ret.checkPlugin(v)
		return true
	})

	// plugin init
	ret.plugin.Range(func(k uuid.UUID, v *sn.SNPluginEntry) bool {
		if v.Enable {
			if err := v.Interface.(plugins.PluginInterface).PluginEnable(); err != nil {
				utils.WithTrace(err).Error(err)
				v.Disable(fmt.Sprintf("Plugin %v enable error: %s", v.Name, err.Error()))
			}
		}
		return true
	})
	return &ret, nil
}

func (p *sitePlugin) Delete(id uuid.UUID) error {
	if err := p.Disable(id); err != nil {
		return err
	}
	item := p.plugin.MustGet(id)
	if err := tracerr.Wrap(os.RemoveAll(item.Path)); err != nil {
		return err
	}
	if err := tracerr.Wrap(os.RemoveAll(path.Join("assets/_plugin", id.String()))); err != nil {
		return err
	}
	if _, err := sn.Skynet.Setting.Delete(settingPrefix + item.ID.String()); err != nil {
		utils.WithTrace(err).Error(err)
	}
	p.plugin.Delete(id)
	return nil
}

func (p *sitePlugin) Update(id uuid.UUID, buf []byte) error {
	pInstance, zipReader, err := p.loadFromByte(buf)
	if err != nil {
		return err
	}
	if pInstance.ID != id {
		return ErrPluginIDNotMatch
	}
	v, exist := p.plugin.Get(id)
	if !exist {
		return ErrPluginNotFound
	}
	if err := tracerr.Wrap(os.RemoveAll(v.Path)); err != nil {
		return err
	}
	if err := tracerr.Wrap(os.RemoveAll(path.Join("assets/_plugin", id.String()))); err != nil {
		return err
	}
	if err = p.UnzipPlugin(v.Path, zipReader); err != nil {
		return err
	}
	os.Rename(path.Join(v.Path, "assets"), path.Join("assets/_plugin", id.String()))
	return nil
}

func (p *sitePlugin) loadFromByte(buf []byte) (*sn.SNPluginInfo, *zip.Reader, error) {
	reader := bytes.NewReader(buf)
	r, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		return nil, nil, tracerr.Wrap(err)
	}
	var ret sn.SNPluginInfo
	if err := tracerr.Wrap(json.Unmarshal([]byte(r.Comment), &ret)); err != nil {
		return nil, nil, err
	}
	return &ret, r, nil
}

func (p *sitePlugin) UnzipPlugin(baseDir string, r *zip.Reader) error {
	fc := func() error {
		if err := tracerr.Wrap(os.Mkdir(baseDir, 0755)); err != nil {
			return err
		}
		for _, f := range r.File {
			if f.FileInfo().IsDir() {
				if err := tracerr.Wrap(os.MkdirAll(path.Join(baseDir, f.Name), 0755)); err != nil {
					return err
				}
			} else {
				out, err := f.Open()
				if err != nil {
					return tracerr.Wrap(err)
				}
				defer out.Close()
				dst, err := os.OpenFile(path.Join(baseDir, f.Name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
				if err != nil {
					return tracerr.Wrap(err)
				}
				defer dst.Close()
				if _, err := io.Copy(dst, out); err != nil {
					return tracerr.Wrap(err)
				}
			}
		}
		return nil
	}

	if err := fc(); err != nil {
		os.RemoveAll(baseDir)
		return err
	}
	return nil
}

func (p *sitePlugin) New(buf []byte) error {
	pInstance, zipReader, err := p.loadFromByte(buf)
	if err != nil {
		return err
	}
	baseDir := path.Join(p.baseFolder, pInstance.Name)
	if utils.FileExist(baseDir) {
		return ErrPluginExists
	}

	if err = p.UnzipPlugin(baseDir, zipReader); err != nil {
		return err
	}
	if err := p.loadPluginFolder(baseDir); err != nil {
		os.RemoveAll(baseDir)
		return err
	}
	os.Rename(path.Join(baseDir, "assets"), path.Join("assets/_plugin", pInstance.ID.String()))
	if err := sn.Skynet.Setting.Set(settingPrefix+pInstance.ID.String(), "0"); err != nil {
		utils.WithTrace(err).Error(err)
	}
	if v, ok := p.plugin.Get(pInstance.ID); ok {
		p.checkPlugin(v)
	}
	return nil
}

func (p *sitePlugin) GetAll() []*sn.SNPluginEntry {
	return p.plugin.Values()
}

func (p *sitePlugin) Get(id uuid.UUID) *sn.SNPluginEntry {
	v, exist := p.plugin.Get(id)
	if !exist {
		return nil
	}
	return v
}

func (p *sitePlugin) Disable(id uuid.UUID) error {
	if v, exist := p.plugin.Get(id); exist {
		if !v.Enable {
			return nil
		}
		if err := v.Interface.(plugins.PluginInterface).PluginDisable(); err != nil {
			return err
		}
		if err := sn.Skynet.Setting.Set(settingPrefix+v.ID.String(), "0"); err != nil {
			utils.WithTrace(err).Error(err)
		}
		v.Enable = false
		return nil
	}
	return ErrPluginNotFound
}

func (p *sitePlugin) Enable(id uuid.UUID) error {
	if v, exist := p.plugin.Get(id); exist {
		if v.Enable {
			return nil
		}
		if !p.checkPlugin(v) {
			return tracerr.New(v.Message)
		}
		if err := v.Interface.(plugins.PluginInterface).PluginEnable(); err != nil {
			return err
		}
		if err := sn.Skynet.Setting.Set(settingPrefix+v.ID.String(), "1"); err != nil {
			utils.WithTrace(err).Error(err)
		}
		v.Enable = true
		return nil
	}
	return ErrPluginNotFound
}

func (p *sitePlugin) Fini() {
	p.plugin.Range(func(k uuid.UUID, v *sn.SNPluginEntry) bool {
		if v.Enable {
			if err := v.Interface.(plugins.PluginInterface).PluginDisable(); err != nil {
				utils.WithTrace(err).Errorf("Plugin %v disable error: %v", v.Name, err)
			}
		}
		return true
	})
}

func (p *sitePlugin) Count() int {
	return p.plugin.Len()
}

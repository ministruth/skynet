package handler

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	sp "github.com/MXWXZ/skynet/plugin"
	"github.com/MXWXZ/skynet/plugin/proto"
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/MXWXZ/skynet/utils/tpl"

	"github.com/google/uuid"
	"github.com/hashicorp/go-plugin"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
	"gopkg.in/yaml.v3"
)

const pluginSettingPrefix = "plugin_"
const pluginConfig = "config.yml"

var Plugin = &PluginImpl{}

var (
	ErrPluginNotFound    = tracerr.New("plugin not found")
	ErrPluginIDDuplicate = tracerr.New("plugin ID duplicated")
	ErrPluginInvalid     = tracerr.New("plugin invalid")
	ErrPluginExists      = tracerr.New("plugin already exists")
	ErrPluginIDNotMatch  = tracerr.New("plugin id not match")
	ErrPluginMethod      = tracerr.New("plugin method error")
)

type PluginSkynetAPI struct {
	Impl sp.PluginAPI
}

func (p *PluginSkynetAPI) Enable() (*sp.PluginError, error) {
	return p.Impl.Enable(&pluginHelper{})
}

func (p *PluginSkynetAPI) Disable() (*sp.PluginError, error) {
	return p.Impl.Disable(&pluginHelper{})
}

type PluginEntry struct {
	ID             uuid.UUID `json:"id" yaml:"id"`                         // plugin unique ID
	Name           string    `json:"name" yaml:"name"`                     // plugin name, unique suggested
	Version        string    `json:"version" yaml:"version"`               // plugin version
	SkynetVersion  string    `json:"skynet_version" yaml:"skynet_version"` // compatible skynet version
	CommandUnix    string    `json:"-" yaml:"command_unix"`                // unix execute command
	CommandWindows string    `json:"-" yaml:"command_windows"`             // windows execute command

	Path    string           `json:"path" yaml:"-"`    // runtime relative path
	Enable  bool             `json:"enable" yaml:"-"`  // is plugin enabled
	Message string           `json:"message" yaml:"-"` // plugin message
	Client  *plugin.Client   `json:"-" yaml:"-"`       // go-plugin client
	API     *PluginSkynetAPI `json:"-" yaml:"-"`       // plugin API
}

func (p *PluginEntry) KillPlugin() {
	if p.Client != nil {
		p.Client.Kill()
	}
}

func (p *PluginEntry) StartPlugin() error {
	if p.Client != nil {
		p.Client.Kill()
	}
	var words []string
	var cmd string
	var args []string
	var err error
	parseString := func(s string) string {
		s = strings.ReplaceAll(s, "$OS", runtime.GOOS)
		s = strings.ReplaceAll(s, "$ARCH", runtime.GOARCH)
		return s
	}
	if runtime.GOOS == "windows" {
		words, err = shellquote.Split(parseString(p.CommandWindows))
	} else {
		words, err = shellquote.Split(parseString(p.CommandUnix))
	}
	if err != nil {
		return err
	}
	if len(words) > 1 {
		cmd = words[0]
		args = words[1:]
	} else {
		cmd = words[0]
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	execCmd := exec.Command(cmd, args...)
	execCmd.Dir = path.Join(root, p.Path)
	p.Client = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  sp.Handshake,
		Plugins:          sp.PluginMap(viper.GetDuration("plugin.timeout") * time.Second),
		Cmd:              execCmd,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           &log.HCLogAdapter{Logger: log.New()},
		SyncStdout:       os.Stdout,
		SyncStderr:       os.Stderr,
	})
	rpc, err := p.Client.Client()
	if err != nil {
		return err
	}
	raw, err := rpc.Dispense("grpc")
	if err != nil {
		return err
	}
	p.API = &PluginSkynetAPI{Impl: raw.(sp.PluginAPI)}
	return nil
}

type PluginImpl struct {
	plugin     *tpl.SafeMap[uuid.UUID, *PluginEntry]
	BaseFolder string
}

func (p *PluginImpl) LoadPlugin(base string) error {
	p.BaseFolder = base
	p.plugin = new(tpl.SafeMap[uuid.UUID, *PluginEntry])

	files, err := ioutil.ReadDir(base)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, f := range files {
		if f.IsDir() {
			pluginPath := path.Join(base, f.Name())
			if err := p.loadFolder(pluginPath); err != nil {
				log.NewEntry(err).WithField("path", pluginPath).Error("Failed to load plugin")
			}
		}
	}
	p.CleanSetting()
	p.plugin.Range(func(k uuid.UUID, v *PluginEntry) bool {
		p.checkPlugin(v)
		return true
	})

	// plugin init
	p.plugin.Range(func(k uuid.UUID, v *PluginEntry) bool {
		if v.Enable {
			ok := true
			var msg string
			if err := v.StartPlugin(); err != nil {
				msg = "Failed to start plugin"
				ok = false
				log.NewEntry(err).WithField("id", v.ID).Error(msg)
			} else {
				rsp, err := v.API.Enable()
				if err != nil {
					msg = "Failed to enable plugin"
					ok = false
					log.NewEntry(err).WithField("id", v.ID).Error(msg)
				}
				if rsp.Code != proto.ErrorCode_OK {
					msg = fmt.Sprintf("Enable plugin return %v", rsp.Code)
					ok = false
					log.NewEntry(ErrPluginMethod).WithField("id", v.ID).Error(msg)
				}
			}
			if !ok {
				v.Enable = false
				v.Message = msg
				if err := Setting.Set(pluginSettingPrefix+v.ID.String(), "0"); err != nil {
					log.NewEntry(err).Error("Failed to update setting")
				}
			}
		}
		return true
	})
	return nil
}

func (p *PluginImpl) load(path string) error {
	config, err := ioutil.ReadFile(filepath.Join(path, "config.yml"))
	if err != nil {
		return tracerr.Wrap(err)
	}
	entry := new(PluginEntry)
	if err = yaml.Unmarshal(config, entry); err != nil {
		return tracerr.Wrap(err)
	}
	if v := p.Get(entry.ID); v != nil {
		return tracerr.Wrap(fmt.Errorf("%w: %v and %v have same ID %v", ErrPluginIDDuplicate, entry.Name, v.Name, v.ID))
	}
	entry.Path = path
	p.plugin.Set(entry.ID, entry)
	log.New().WithFields(log.F{
		"id":      entry.ID,
		"name":    entry.Name,
		"version": entry.Version,
	}).Info("Plugin loaded")
	return nil
}

func (p *PluginImpl) loadFolder(dir string) error {
	dirFile, err := ioutil.ReadDir(dir)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, df := range dirFile {
		if df.Name() == pluginConfig {
			if err := p.load(dir); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *PluginImpl) checkPlugin(v *PluginEntry) bool {
	// check version
	c, err := utils.CheckVersion(sn.Version, v.SkynetVersion)
	if err != nil {
		log.NewEntry(err).WithFields(log.F{
			"id":         v.ID,
			"constraint": v.SkynetVersion,
		}).Error("Version constraint invalid")
	}
	if !c {
		v.Enable = false
		v.Message = fmt.Sprintf("Skynet version mismatch, need %s", v.SkynetVersion)
		log.New().WithFields(log.F{
			"version":    sn.Version,
			"constraint": v.SkynetVersion,
			"id":         v.ID,
		}).Error("Plugin skynet version mismatch, disable now.")
		if err := Setting.Set(pluginSettingPrefix+v.ID.String(), "0"); err != nil {
			log.NewEntry(err).Error("Failed to update setting")
		}
		return false
	}

	return true
}

func (p *PluginImpl) CleanSetting() {
	setting := Setting.GetAll()
	// setting enable cleanup
	for k, v := range setting {
		if strings.HasPrefix(k, pluginSettingPrefix) && v == "1" {
			setting[k] = "-1"
		}
	}

	p.plugin.Range(func(k uuid.UUID, v *PluginEntry) bool {
		name := pluginSettingPrefix + v.ID.String()
		if status, ok := setting[name]; ok {
			v.Enable = status == "-1"
			if v.Enable {
				setting[name] = "1"
			}
		} else {
			if err := Setting.Set(name, "0"); err != nil {
				log.NewEntry(err).Error("Failed to update setting")
			}
		}
		return true
	})

	for k, v := range setting {
		if strings.HasPrefix(k, pluginSettingPrefix) && v == "-1" {
			if _, err := Setting.Delete(k); err != nil {
				log.NewEntry(err).Error("Failed to delete setting")
			}
		}
	}
}

// func (p *PluginImpl) New(buf []byte) error {
// 	pInstance, zipReader, err := p.loadFromByte(buf)
// 	if err != nil {
// 		return err
// 	}
// 	baseDir := path.Join(p.baseFolder, pInstance.Name)
// 	if utils.FileExist(baseDir) {
// 		return ErrPluginExists
// 	}

// 	if err = p.UnzipPlugin(baseDir, zipReader); err != nil {
// 		return err
// 	}
// 	if err := p.loadPluginFolder(baseDir); err != nil {
// 		os.RemoveAll(baseDir)
// 		return err
// 	}
// 	os.Rename(path.Join(baseDir, "assets"), path.Join("assets/_plugin", pInstance.ID.String()))
// 	if err := sn.Skynet.Setting.Set(settingPrefix+pInstance.ID.String(), "0"); err != nil {
// 		utils.WithTrace(err).Error(err)
// 	}
// 	if v, ok := p.plugin.Get(pInstance.ID); ok {
// 		p.checkPlugin(v)
// 	}
// 	return nil
// }

// func (p *PluginImpl) Call(cb sn.SNPluginCBType, param interface{}) []error {
// 	var ret []error = nil
// 	p.plugin.Range(func(k uuid.UUID, v *sn.SNPluginEntry) bool {
// 		if v.Enable && v.Callback != nil {
// 			if f, ok := v.Callback[cb]; ok && f != nil {
// 				if err := f(param); err != nil {
// 					ret = append(ret, err)
// 				}
// 			}
// 		}
// 		return true
// 	})
// 	return ret
// }

// func (p *PluginImpl) Delete(id uuid.UUID) error {
// 	if err := p.Disable(id); err != nil {
// 		return err
// 	}
// 	item := p.plugin.MustGet(id)
// 	if err := tracerr.Wrap(os.RemoveAll(item.Path)); err != nil {
// 		return err
// 	}
// 	if err := tracerr.Wrap(os.RemoveAll(path.Join("assets/_plugin", id.String()))); err != nil {
// 		return err
// 	}
// 	if _, err := sn.Skynet.Setting.Delete(settingPrefix + item.ID.String()); err != nil {
// 		utils.WithTrace(err).Error(err)
// 	}
// 	p.plugin.Delete(id)
// 	return nil
// }

// func (p *PluginImpl) Update(id uuid.UUID, buf []byte) error {
// 	pInstance, zipReader, err := p.loadFromByte(buf)
// 	if err != nil {
// 		return err
// 	}
// 	if pInstance.ID != id {
// 		return ErrPluginIDNotMatch
// 	}
// 	v, exist := p.plugin.Get(id)
// 	if !exist {
// 		return ErrPluginNotFound
// 	}
// 	if err := tracerr.Wrap(os.RemoveAll(v.Path)); err != nil {
// 		return err
// 	}
// 	if err := tracerr.Wrap(os.RemoveAll(path.Join("assets/_plugin", id.String()))); err != nil {
// 		return err
// 	}
// 	if err = p.UnzipPlugin(v.Path, zipReader); err != nil {
// 		return err
// 	}
// 	os.Rename(path.Join(v.Path, "assets"), path.Join("assets/_plugin", id.String()))
// 	return nil
// }

// func (p *PluginImpl) loadFromByte(buf []byte) (*sn.SNPluginInfo, *zip.Reader, error) {
// 	reader := bytes.NewReader(buf)
// 	r, err := zip.NewReader(reader, reader.Size())
// 	if err != nil {
// 		return nil, nil, tracerr.Wrap(err)
// 	}
// 	var ret sn.SNPluginInfo
// 	if err := tracerr.Wrap(json.Unmarshal([]byte(r.Comment), &ret)); err != nil {
// 		return nil, nil, err
// 	}
// 	return &ret, r, nil
// }

// func (p *PluginImpl) UnzipPlugin(baseDir string, r *zip.Reader) error {
// 	fc := func() error {
// 		if err := tracerr.Wrap(os.Mkdir(baseDir, 0755)); err != nil {
// 			return err
// 		}
// 		for _, f := range r.File {
// 			if f.FileInfo().IsDir() {
// 				if err := tracerr.Wrap(os.MkdirAll(path.Join(baseDir, f.Name), 0755)); err != nil {
// 					return err
// 				}
// 			} else {
// 				out, err := f.Open()
// 				if err != nil {
// 					return tracerr.Wrap(err)
// 				}
// 				defer out.Close()
// 				dst, err := os.OpenFile(path.Join(baseDir, f.Name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
// 				if err != nil {
// 					return tracerr.Wrap(err)
// 				}
// 				defer dst.Close()
// 				if _, err := io.Copy(dst, out); err != nil {
// 					return tracerr.Wrap(err)
// 				}
// 			}
// 		}
// 		return nil
// 	}

// 	if err := fc(); err != nil {
// 		os.RemoveAll(baseDir)
// 		return err
// 	}
// 	return nil
// }

func (p *PluginImpl) GetAll() []*PluginEntry {
	return p.plugin.Values()
}

func (p *PluginImpl) Get(id uuid.UUID) *PluginEntry {
	v, exist := p.plugin.Get(id)
	if !exist {
		return nil
	}
	return v
}

func (p *PluginImpl) Disable(id uuid.UUID) error {
	if v, exist := p.plugin.Get(id); exist {
		if !v.Enable {
			return nil
		}
		rsp, err := v.API.Disable()
		if err != nil {
			return err
		}
		if rsp.Code != proto.ErrorCode_OK {
			return fmt.Errorf("%w: %v", ErrPluginMethod, rsp.Code)
		}
		if err := Setting.Set(pluginSettingPrefix+v.ID.String(), "0"); err != nil {
			log.NewEntry(err).Error("Failed to update setting")
		}
		v.KillPlugin()
		v.Enable = false
		return nil
	}
	return ErrPluginNotFound
}

func (p *PluginImpl) Enable(id uuid.UUID) error {
	if v, exist := p.plugin.Get(id); exist {
		if v.Enable {
			return nil
		}
		if !p.checkPlugin(v) {
			return tracerr.New(v.Message)
		}
		if err := v.StartPlugin(); err != nil {
			return err
		}
		rsp, err := v.API.Enable()
		if err != nil {
			v.KillPlugin()
			return err
		}
		if rsp.Code != proto.ErrorCode_OK {
			v.KillPlugin()
			return fmt.Errorf("%w: %v", ErrPluginMethod, rsp.Code)
		}
		if err := Setting.Set(pluginSettingPrefix+v.ID.String(), "1"); err != nil {
			log.NewEntry(err).Error("Failed to update setting")
		}
		v.Enable = true
		return nil
	}
	return ErrPluginNotFound
}

func (p *PluginImpl) Fini() {
	p.plugin.Range(func(k uuid.UUID, v *PluginEntry) bool {
		if v.Enable {
			rsp, err := v.API.Disable()
			if err != nil {
				log.NewEntry(err).Errorf("Failed to disable plugin %v", v.Name)
			}
			if rsp.Code != proto.ErrorCode_OK {
				log.New().Errorf("Disable plugin %v return %v", v.Name, rsp.Code)
			}
			v.Client.Kill()
		}
		return true
	})
}

func (p *PluginImpl) Count() int {
	return p.plugin.Len()
}

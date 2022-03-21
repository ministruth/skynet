package plugin

import (
	"embed"
	"io/fs"
	"path"
	"skynet/sn"
	"skynet/sn/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

type PluginInfo struct {
	sn.SNPluginInfo
}

// PluginInterface is plugin interface, every plugin should export one with NewPlugin function.
// Signature: func NewPlugin() PluginInterface
type PluginInterface interface {
	// Instance will return the instance of the plugin.
	// You should make sure the instance returned will be persist and the same each time calls.
	Instance() *sn.SNPluginInfo

	// PluginEnable will be executed when plugin enabled or loaded,
	// return error to stop plugin enable.
	//
	// Note: the plugin initialize order may not stable.
	PluginEnable() error

	// PluginDisable will be executed when plugin disabled/uninstalled or skynet exit.
	// return error to stop plugin disable.
	//
	// Note: the plugin exit order may not stable.
	PluginDisable() error
}

func (p *PluginInfo) Log() *log.Entry {
	return p.LogF(nil, uuid.Nil, nil)
}

// Log return logrus instance.
func (p *PluginInfo) LogF(c *gin.Context, u uuid.UUID, f log.Fields) *log.Entry {
	ret := log.WithField("plugin", p.ID).WithFields(f)
	if c != nil {
		tmp := utils.MustMarshal(c.Request.URL.Query())
		if tmp != "{}" {
			ret = ret.WithField("param", tmp)
		}
		ret = ret.WithField("ip", utils.GetIP(c))
	}
	if u != uuid.Nil {
		ret = ret.WithField("user", u)
	}
	return ret
}

// Response is wrapper for c.JSON.
func (p *PluginInfo) Response(c *gin.Context, code int, msg string, data any, other gin.H) {
	ret := gin.H{"code": code, "msg": msg}
	if data != nil {
		ret["data"] = data
	}
	for k, v := range other {
		ret[k] = v
	}

	c.JSON(200, ret)
}

// ResponseOK is wrapper for ok response.
func (p *PluginInfo) ResponseOK(c *gin.Context) {
	p.Response(c, 0, "Success", nil, nil)
}

// ResponseData is wrapper for data response.
func (p *PluginInfo) ResponseData(c *gin.Context, data any) {
	p.Response(c, 0, "Success", data, nil)
}

// ResponsePage is wrapper for page response.
func (p *PluginInfo) ResponsePage(c *gin.Context, data any, total any) {
	p.Response(c, 0, "Success", data, gin.H{"total": total})
}

// LogSuccess log and add success notification.
func (p *PluginInfo) LogSuccess(l *log.Entry, s string) {
	if l == nil {
		l = p.Log()
	}
	l.Info(s)
	sn.Skynet.Notification.New(sn.NotifySuccess, "Plugin log", s, utils.MustMarshal(l.Data))
}

// GetPath return runtime path.
func (p *PluginInfo) GetPath() string {
	return sn.Skynet.Plugin.Get(p.ID).Path
}

// GetTempFilePath returns the relative temp file path with suffix.
func (p *PluginInfo) GetTempFilePath(suffix string) string {
	return path.Join("temp/plugin", p.ID.String(), suffix)
}

// GetDataFilePath returns the relative data file path with suffix.
func (p *PluginInfo) GetDataFilePath(suffix string) string {
	return path.Join("data/plugin", p.ID.String(), suffix)
}

// AddPluginMenu add plugin menu.
func (p *PluginInfo) AddPluginMenu(menu *sn.SNMenu) bool {
	return sn.Skynet.API.AddMenu(menu, uuid.MustParse("cca5b3b0-40a3-465c-8b08-91f3e8d3b14d"))
}

// AddServiceMenu add service menu.
func (p *PluginInfo) AddServiceMenu(menu *sn.SNMenu) bool {
	return sn.Skynet.API.AddMenu(menu, uuid.MustParse("d00d36d0-6068-4447-ab04-f82ce893c04e"))
}

// InitPermission get or create default permission.
func (p *PluginInfo) InitPermission(list *sn.PermissionList) error {
	return tracerr.Wrap(sn.Skynet.GetDB().Where(&sn.PermissionList{Name: list.Name}).
		Attrs(&sn.PermissionList{Note: list.Note}).
		FirstOrCreate(list).Error)
}

// ParseLang parse language translation from embed fs.
func (p *PluginInfo) ParseLang(f embed.FS) error {
	return fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return tracerr.Wrap(err)
		}
		if !d.IsDir() {
			b, err := f.ReadFile("i18n/" + d.Name())
			if err != nil {
				return tracerr.Wrap(err)
			}
			_, err = sn.Skynet.Translator.ParseMessageFileBytes(b, d.Name())
			if err != nil {
				return tracerr.Wrap(err)
			}
		}
		return nil
	})
}

type PaginationParam struct {
	Page int `form:"page,default=1" binding:"min=1"`
	Size int `form:"size,default=10" binding:"min=1"`
}

type CreatedParam struct {
	CreatedSort  string `form:"createdSort" binding:"omitempty,oneof=asc desc"`
	CreatedStart int64  `form:"createdStart" binding:"min=0"`
	CreatedEnd   int64  `form:"createdEnd" binding:"min=0"`
}

type UpdatedParam struct {
	UpdatedSort  string `form:"updatedSort" binding:"omitempty,oneof=asc desc"`
	UpdatedStart int64  `form:"updatedStart" binding:"min=0"`
	UpdatedEnd   int64  `form:"updatedEnd" binding:"min=0"`
}

type IDURI struct {
	ID string `uri:"id" binding:"required,uuid"`
}

func (p *PluginInfo) BuildCondition(created *CreatedParam, updated *UpdatedParam,
	pageParam *PaginationParam, text string, condText string) *sn.SNCondition {
	now := time.Now().UnixMilli()
	if created != nil && created.CreatedEnd == 0 {
		created.CreatedEnd = now
	}
	if updated != nil && updated.UpdatedEnd == 0 {
		updated.UpdatedEnd = now
	}

	cond := &sn.SNCondition{
		Limit:  pageParam.Size,
		Offset: (pageParam.Page - 1) * pageParam.Size,
	}

	if created != nil && !(created.CreatedStart == 0 && created.CreatedEnd == now) {
		cond.And("created_at BETWEEN ? AND ?", created.CreatedStart, created.CreatedEnd)
	}
	if updated != nil && !(updated.UpdatedStart == 0 && updated.UpdatedEnd == now) {
		cond.And("updated_at BETWEEN ? AND ?", updated.UpdatedStart, updated.UpdatedEnd)
	}
	if text != "" {
		cond.And(condText)
		count := strings.Count(condText, "?")
		for i := 0; i < count; i++ {
			cond.Args = append(cond.Args, "%"+text+"%")
		}
	}
	if updated != nil && updated.UpdatedSort != "" {
		cond.Order = []any{"updated_at " + updated.UpdatedSort}
	}
	if created != nil && created.CreatedSort != "" {
		cond.Order = []any{"created_at " + created.CreatedSort}
	}
	return cond
}

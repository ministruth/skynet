package api

import (
	"errors"
	"skynet/handler"
	"skynet/sn"
	"skynet/sn/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

func APIGetPluginEntry(c *gin.Context, id uuid.UUID) (int, error) {
	ret, err := GetMenuPluginID(c)
	if err != nil {
		return 500, err
	}
	responseData(c, ret)
	return 0, nil
}

func APIGetPlugin(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		Enable []bool `form:"enable[]"`
		Text   string `form:"text"`
		paginationParam
	}
	var param Param
	if err := tracerr.Wrap(c.ShouldBindQuery(&param)); err != nil {
		return 400, err
	}

	if len(param.Enable) == 0 {
		param.Enable = []bool{
			false,
			true,
		}
	}
	contains := func(v bool) bool {
		for _, e := range param.Enable {
			if e == v {
				return true
			}
		}
		return false
	}

	p := sn.Skynet.Plugin.GetAll()
	var ret []*sn.SNPluginEntry
	for _, v := range p {
		if contains(v.Enable) {
			if param.Text == "" {
				ret = append(ret, v)
			} else {
				if strings.Contains(v.ID.String(), param.Text) ||
					strings.Contains(v.Name, param.Text) ||
					strings.Contains(v.Message, param.Text) {
					ret = append(ret, v)
				}
			}
		}
	}
	min, max, ok := utils.CalcPage(param.Page, param.Size, len(ret))
	if !ok {
		responsePage(c, []*sn.SNPluginEntry{}, len(ret))
	} else {
		responsePage(c, ret[min:max], len(ret))
	}
	return 0, nil
}

func APIPutPlugin(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		Enable bool `json:"enable"`
	}

	var param Param
	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	var uriParam idURI
	if err := tracerr.Wrap(c.ShouldBindUri(&uriParam)); err != nil {
		return 400, err
	}
	pluginID, err := uuid.Parse(uriParam.ID)
	if err != nil {
		return 400, err
	}
	logf := wrap(c, id, log.Fields{
		"pluginID": pluginID,
		"enable":   param.Enable,
	})

	if sn.Skynet.Plugin.Get(pluginID) == nil {
		logf.Warn(CodePluginNotExist.String(c))
		response(c, CodePluginNotExist)
		return 0, nil
	}

	if param.Enable {
		if err := sn.Skynet.Plugin.Enable(pluginID); err != nil {
			return 500, err
		}
		success(logf, "Enable plugin")
		response(c, CodeOK)
	} else {
		if err := sn.Skynet.Plugin.Disable(pluginID); err != nil {
			return 500, err
		}
		success(logf, "Disable plugin")
		response(c, CodeOK)
		sn.Skynet.Running = false
		go func() {
			time.Sleep(time.Second * 2)
			utils.Restart()
		}()
	}
	return 0, nil
}

func APIDeletePlugin(c *gin.Context, id uuid.UUID) (int, error) {
	var uriParam idURI
	if err := tracerr.Wrap(c.ShouldBindUri(&uriParam)); err != nil {
		return 400, err
	}
	pluginID, err := uuid.Parse(uriParam.ID)
	if err != nil {
		return 400, err
	}
	logf := wrap(c, id, log.Fields{
		"pluginID": pluginID,
	})

	p := sn.Skynet.Plugin.Get(pluginID)
	if p == nil {
		logf.Warn(CodePluginNotExist.String(c))
		response(c, CodePluginNotExist)
		return 0, nil
	}
	if err := sn.Skynet.Plugin.Delete(pluginID); err != nil {
		return 500, err
	}
	success(logf, "Delete plugin")
	response(c, CodeOK)
	if p.Enable {
		sn.Skynet.Running = false
		go func() {
			time.Sleep(time.Second * 2)
			utils.Restart()
		}()
	}
	return 0, nil
}

func APIAddPlugin(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		File []byte `json:"file"`
	}
	var param Param
	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	logf := wrap(c, id, nil)

	if err := sn.Skynet.Plugin.New(param.File); err != nil {
		if errors.Is(err, handler.ErrPluginInvalid) {
			logf.Warn(CodePluginFormatError.String(c))
			response(c, CodePluginFormatError)
			return 0, nil
		}
		if errors.Is(err, handler.ErrPluginExists) {
			logf.Warn(CodePluginExist.String(c))
			response(c, CodePluginExist)
			return 0, nil
		}
		return 500, err
	}
	success(logf, "Add plugin")
	response(c, CodeOK)
	return 0, nil
}

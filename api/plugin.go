package api

import (
	"strings"

	"github.com/MXWXZ/skynet/handler"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"

	"github.com/google/uuid"
)

func APIGetPluginEntry(req *Request) (*Response, error) {
	ret, err := GetMenuPluginID(req)
	if err != nil {
		return nil, err
	}
	return &Response{Data: ret}, nil
}

func APIGetPlugin(req *Request) (*Response, error) {
	type Param struct {
		Enable []bool `form:"enable[]"`
		Text   string `form:"text"`
		paginationParam
	}
	var param Param
	if err := req.ShouldBindQuery(&param); err != nil {
		return rspParamInvalid, err
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

	p := handler.Plugin.GetAll()
	var ret []*handler.PluginEntry
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
		return NewPageResponse([]*handler.PluginEntry{}, int64(len(ret))), nil
	} else {
		return NewPageResponse(ret[min:max], int64(len(ret))), nil
	}
}

func APIPutPlugin(req *Request) (*Response, error) {
	type Param struct {
		Enable bool `json:"enable"`
	}

	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return rspParamInvalid, err
	}
	var uriParam idURI
	if err := req.ShouldBindUri(&uriParam); err != nil {
		return rspParamInvalid, err
	}
	pluginID, err := uuid.Parse(uriParam.ID)
	if err != nil {
		return rspParamInvalid, err
	}
	logger := req.Logger.WithFields(log.F{
		"pluginID": pluginID,
		"enable":   param.Enable,
	})

	if handler.Plugin.Get(pluginID) == nil {
		logger.Warn(CodePluginNotExist.String(req.Translator))
		return &Response{Code: CodePluginNotExist}, nil
	}

	if param.Enable {
		if err := handler.Plugin.Enable(pluginID); err != nil {
			return nil, err
		}
		success(logger, "Enable plugin")
	} else {
		if err := handler.Plugin.Disable(pluginID); err != nil {
			return nil, err
		}
		success(logger, "Disable plugin")
	}
	return rspOK, nil
}

// func APIDeletePlugin(c *gin.Context, id uuid.UUID) (int, error) {
// 	var uriParam idURI
// 	if err := tracerr.Wrap(c.ShouldBindUri(&uriParam)); err != nil {
// 		return 400, err
// 	}
// 	pluginID, err := uuid.Parse(uriParam.ID)
// 	if err != nil {
// 		return 400, err
// 	}
// 	logf := wrap(c, id, log.Fields{
// 		"pluginID": pluginID,
// 	})

// 	p := sn.Skynet.Plugin.Get(pluginID)
// 	if p == nil {
// 		logf.Warn(CodePluginNotExist.String(c))
// 		response(c, CodePluginNotExist)
// 		return 0, nil
// 	}
// 	if err := sn.Skynet.Plugin.Delete(pluginID); err != nil {
// 		return 500, err
// 	}
// 	success(logf, "Delete plugin")
// 	response(c, CodeOK)
// 	if p.Enable {
// 		sn.Skynet.Running = false
// 		go func() {
// 			time.Sleep(time.Second * 2)
// 			utils.Restart()
// 		}()
// 	}
// 	return 0, nil
// }

// func APIAddPlugin(c *gin.Context, id uuid.UUID) (int, error) {
// 	type Param struct {
// 		File []byte `json:"file"`
// 	}
// 	var param Param
// 	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
// 		return 400, err
// 	}
// 	logf := wrap(c, id, nil)

// 	if err := sn.Skynet.Plugin.New(param.File); err != nil {
// 		if errors.Is(err, handler.ErrPluginInvalid) {
// 			logf.Warn(CodePluginFormatError.String(c))
// 			response(c, CodePluginFormatError)
// 			return 0, nil
// 		}
// 		if errors.Is(err, handler.ErrPluginExists) {
// 			logf.Warn(CodePluginExist.String(c))
// 			response(c, CodePluginExist)
// 			return 0, nil
// 		}
// 		return 500, err
// 	}
// 	success(logf, "Add plugin")
// 	response(c, CodeOK)
// 	return 0, nil
// }

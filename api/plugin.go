package api

import (
	"strings"

	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

func APIGetPluginEntry(req *sn.Request) (*sn.Response, error) {
	return &sn.Response{Data: []uuid.UUID{}}, nil
}

func APIGetPlugin(req *sn.Request) (*sn.Response, error) {
	type PluginStatus = int32
	const (
		PluginUnload PluginStatus = iota
		PluginDisable
		PluginEnable
	)
	type Param struct {
		Status []PluginStatus `form:"status[]" binding:"dive,min=0,max=2"`
		Text   string         `form:"text"`
		sn.PaginationParam
	}
	type Rsp struct {
		ID      uuid.UUID    `json:"id"`
		Name    string       `json:"name"`
		Version string       `json:"version"`
		Path    string       `json:"path"`
		Status  PluginStatus `json:"status"`
	}
	var param Param
	if err := req.ShouldBindQuery(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}

	if len(param.Status) == 0 {
		param.Status = []PluginStatus{
			PluginUnload,
			PluginDisable,
			PluginEnable,
		}
	}

	ret := []*Rsp{}
	p := sn.Skynet.Plugin.GetAll()
	for _, v := range p {
		var status PluginStatus
		if v.Instance == nil {
			status = PluginUnload
		} else if !v.Enable {
			status = PluginDisable
		} else {
			status = PluginEnable
		}
		if slices.Contains(param.Status, status) {
			if param.Text != "" {
				if !strings.Contains(v.ID.String(), param.Text) &&
					!strings.Contains(v.Name, param.Text) {
					continue
				}
			}
			// filtered
			ret = append(ret, &Rsp{
				ID:      v.ID,
				Name:    v.Name,
				Version: v.Version,
				Path:    v.Path,
				Status:  status,
			})
		}
	}

	return sn.NewPageResponse(utils.SlicePagination(ret, param.Page, param.Size), int64(len(ret))), nil
}

func pluginWrapper(req *sn.Request) (*sn.PluginEntry, *sn.Response, error) {
	var uriParam sn.IDURI
	if err := req.ShouldBindUri(&uriParam); err != nil {
		return nil, sn.ResponseParamInvalid, err
	}
	pid, err := uriParam.Parse()
	if err != nil {
		return nil, sn.ResponseParamInvalid, err
	}

	ret := sn.Skynet.Plugin.Get(pid)
	if ret == nil {
		return nil, &sn.Response{Code: sn.CodePluginNotExist}, nil
	}
	return ret, nil, nil
}

func APIPutPlugin(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Enable bool `json:"enable"`
	}

	entry, ret, err := pluginWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}

	if param.Enable != entry.Enable {
		if param.Enable {
			if err := sn.Skynet.Plugin.Enable(entry.ID); err != nil {
				return nil, err
			}
		} else {
			if err := sn.Skynet.Plugin.Disable(entry.ID); err != nil {
				return nil, err
			}
		}
	}
	return sn.ResponseOK, nil
}

func APIDeletePlugin(req *sn.Request) (*sn.Response, error) {
	entry, ret, err := pluginWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	if entry.Instance != nil {
		return &sn.Response{Code: sn.CodePluginLoaded}, nil
	}
	if err := sn.Skynet.Plugin.Delete(entry.ID); err != nil {
		return nil, err
	}
	logger := req.Logger.WithFields(log.F{
		"pluginID": entry.ID,
	})
	success(logger, "Delete plugin")
	return sn.ResponseOK, nil
}

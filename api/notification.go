package api

import (
	"github.com/MXWXZ/skynet/sn"
)

func APIGetUnreadNotification(req *sn.Request) (*sn.Response, error) {
	return &sn.Response{Data: sn.Skynet.Notification.GetUnread()}, nil
}

func APIGetNotification(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Level []sn.NotifyLevel `form:"level[]" binding:"dive,min=0,max=4"`
		Text  string           `form:"text"`
		sn.CreatedParam
		sn.PaginationParam
	}
	var param Param
	if err := req.ShouldBindQuery(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}

	if len(param.Level) == 0 {
		param.Level = []sn.NotifyLevel{
			sn.NotifyInfo,
			sn.NotifySuccess,
			sn.NotifyWarning,
			sn.NotifyError,
			sn.NotifyFatal,
		}
	}

	cond := new(sn.Condition)
	cond.MergeAnd(param.PaginationParam.ToCondition())
	cond.MergeAnd(param.CreatedParam.ToCondition())
	if param.Text != "" {
		cond.AndLike("id LIKE ? OR name LIKE ? OR message LIKE ? OR detail LIKE ?", param.Text)
	}
	cond.And("level IN ?", param.Level)

	rec, err := sn.Skynet.Notification.GetAll(cond)
	if err != nil {
		return nil, err
	}
	count, err := sn.Skynet.Notification.Count(cond)
	if err != nil {
		return nil, err
	}
	sn.Skynet.Notification.SetUnread(0)

	return sn.NewPageResponse(rec, count), nil
}

func APIDeleteNotification(req *sn.Request) (*sn.Response, error) {
	if _, err := sn.Skynet.Notification.DeleteAll(); err != nil {
		return nil, err
	}
	success(req.Logger, "Delete all notification")
	return sn.ResponseOK, nil
}

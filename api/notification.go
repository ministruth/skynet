package api

import (
	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/handler"
)

func APIGetNotification(req *Request) (*Response, error) {
	type Param struct {
		Level []db.NotifyLevel `form:"level[]" binding:"dive,min=0,max=4"`
		Text  string           `form:"text"`
		createdParam
		paginationParam
	}
	var param Param
	if err := req.ShouldBindQuery(&param); err != nil {
		return rspParamInvalid, err
	}

	if len(param.Level) == 0 {
		param.Level = []db.NotifyLevel{
			db.NotifyInfo,
			db.NotifySuccess,
			db.NotifyWarning,
			db.NotifyError,
			db.NotifyFatal,
		}
	}

	cond := new(db.Condition)
	cond.MergeAnd(param.paginationParam.ToCondition())
	cond.MergeAnd(param.createdParam.ToCondition())
	cond.AndLike("id LIKE ? OR name LIKE ? OR message LIKE ? OR detail LIKE ?", param.Text)
	cond.And("level IN ?", param.Level)

	rec, err := handler.Notification.GetAll(cond)
	if err != nil {
		return nil, err
	}
	count, err := handler.Notification.Count(cond)
	if err != nil {
		return nil, err
	}

	return NewPageResponse(rec, count), nil
}

func APIDeleteNotification(req *Request) (*Response, error) {
	if _, err := handler.Notification.DeleteAll(); err != nil {
		return nil, err
	}
	success(req.Logger, "Delete all notification")
	return rspOK, nil
}

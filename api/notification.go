package api

import (
	"skynet/sn"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
)

func APIGetNotification(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		Level []sn.NotifyLevel `form:"level[]" binding:"dive,min=0,max=4"`
		Text  string           `form:"text"`
		createdParam
		paginationParam
	}
	var param Param
	if err := tracerr.Wrap(c.ShouldBindQuery(&param)); err != nil {
		return 400, err
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

	cond := buildCondition(&param.createdParam, nil, &param.paginationParam,
		param.Text, "id LIKE ? OR name LIKE ? OR message LIKE ? OR detail LIKE ?")
	cond.And("level IN ?", param.Level)

	rec, err := sn.Skynet.Notification.GetAll(cond)
	if err != nil {
		return 500, err
	}
	count, err := sn.Skynet.Notification.Count(cond)
	if err != nil {
		return 500, err
	}
	responsePage(c, rec, count)
	return 0, nil
}

func APIDeleteNotification(c *gin.Context, id uuid.UUID) (int, error) {
	logf := wrap(c, id, nil)
	if _, err := sn.Skynet.Notification.DeleteAll(); err != nil {
		return 500, err
	}
	success(logf, "Delete all notification")
	response(c, CodeOK)
	return 0, nil
}

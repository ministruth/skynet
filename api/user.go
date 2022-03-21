package api

import (
	"io/ioutil"
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

func APIGetUser(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		Text           string `form:"text"`
		LastLoginSort  string `form:"lastLoginSort" binding:"omitempty,oneof=asc desc"`
		LastLoginStart int64  `form:"lastLoginStart" binding:"min=0"`
		LastLoginEnd   int64  `form:"lastLoginEnd" binding:"min=0"`
		createdParam
		paginationParam
	}
	var param Param
	if err := tracerr.Wrap(c.ShouldBindQuery(&param)); err != nil {
		return 400, err
	}

	cond := buildCondition(&param.createdParam, nil, &param.paginationParam,
		param.Text, "username LIKE ? OR last_ip LIKE ?")
	now := time.Now().UnixMilli()
	if param.LastLoginEnd == 0 {
		param.LastLoginEnd = now
	}
	if !(param.LastLoginStart == 0 && param.LastLoginEnd == now) {
		cond.And("last_login BETWEEN ? AND ?", param.LastLoginStart,
			param.LastLoginEnd)
	}
	if param.LastLoginSort != "" {
		cond.Order = []any{"last_login " + param.LastLoginSort}
	}

	rec, err := sn.Skynet.User.GetAll(cond)
	if err != nil {
		return 500, err
	}
	count, err := sn.Skynet.User.Count(cond)
	if err != nil {
		return 500, err
	}
	responsePage(c, rec, count)
	return 0, nil
}

func APIGetGroup(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		Text string `form:"text"`
		createdParam
		updatedParam
		paginationParam
	}
	var param Param
	if err := tracerr.Wrap(c.ShouldBindQuery(&param)); err != nil {
		return 400, err
	}

	cond := buildCondition(&param.createdParam, &param.updatedParam, &param.paginationParam,
		param.Text, "name LIKE ? OR note LIKE ?")

	rec, err := sn.Skynet.Group.GetAll(cond)
	if err != nil {
		return 500, err
	}
	count, err := sn.Skynet.Group.Count(cond)
	if err != nil {
		return 500, err
	}
	responsePage(c, rec, count)
	return 0, nil
}

type groupParam struct {
	Name string `json:"name" binding:"required"`
	Note string `json:"note"`
}

func APIAddGroup(c *gin.Context, id uuid.UUID) (int, error) {
	type Param struct {
		groupParam
		Base      string `json:"base" binding:"omitempty,uuid"`
		CloneUser bool   `json:"clone_user"`
	}
	var param Param
	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	var baseID uuid.UUID
	if param.Base != "" {
		baseID = uuid.MustParse(param.Base)
	}
	logf := wrap(c, id, log.Fields{
		"name":      param.Name,
		"note":      param.Note,
		"base":      param.Base,
		"cloneUser": param.CloneUser,
	})

	var group *sn.UserGroup
	if err := sn.Skynet.GetDB().Transaction(func(tx *gorm.DB) error {
		ok, err := sn.Skynet.Group.WithTx(tx).GetByName(param.Name)
		if err != nil {
			return err
		}
		if ok != nil {
			logf.Warn(CodeGroupNameExist.String(c))
			response(c, CodeGroupNameExist)
			return nil
		}
		group, err = sn.Skynet.Group.WithTx(tx).New(param.Name, param.Note)
		if err != nil {
			return err
		}
		if baseID != uuid.Nil {
			base, err := sn.Skynet.Permission.WithTx(tx).GetAll(uuid.Nil, baseID, false)
			if err != nil {
				return err
			}
			var perm []*sn.SNPerm
			for _, v := range base {
				perm = append(perm, &sn.SNPerm{
					ID:   v.PID,
					Perm: v.Perm,
				})
			}
			_, err = sn.Skynet.Permission.WithTx(tx).AddToGroup(group.ID, perm)
			if err != nil {
				return err
			}
		}
		if param.CloneUser {
			user, err := sn.Skynet.Group.WithTx(tx).GetGroupAllUser(baseID)
			if err != nil {
				return err
			}
			var uid []uuid.UUID
			for _, v := range user {
				uid = append(uid, v.ID)
			}
			_, err = sn.Skynet.Group.WithTx(tx).Link(uid, []uuid.UUID{group.ID})
			if err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return 500, err
	}
	if group != nil {
		logf = logf.WithField("gid", group.ID)
		success(logf, "Add user group")
		responseData(c, group.ID)
	}
	return 0, nil
}

func APIPutGroup(c *gin.Context, id uuid.UUID) (int, error) {
	var uriParam idURI
	if err := tracerr.Wrap(c.ShouldBindUri(&uriParam)); err != nil {
		return 400, err
	}
	gid, err := uuid.Parse(uriParam.ID)
	if err != nil {
		return 400, err
	}
	var param groupParam
	if err := tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	logf := wrap(c, id, log.Fields{
		"gid":  gid,
		"name": param.Name,
		"note": param.Note,
	})

	if gid == sn.Skynet.GetID(sn.GroupRootID) && param.Name != "root" {
		logf.Warn(CodeRootNotAllowedUpdate)
		response(c, CodeRootNotAllowedUpdate)
		return 0, nil
	}
	group, err := sn.Skynet.Group.Get(gid)
	if err != nil {
		return 500, err
	}
	if group == nil {
		logf.Warn(CodeGroupNotExist.String(c))
		response(c, CodeGroupNotExist)
		return 0, nil
	}
	if param.Name == group.Name && param.Note == group.Note {
		response(c, CodeOK)
		return 0, nil
	}
	if param.Name != group.Name {
		newGroup, err := sn.Skynet.Group.GetByName(param.Name)
		if err != nil {
			return 500, err
		}
		if newGroup != nil {
			logf.Warn(CodeGroupNameExist)
			response(c, CodeGroupNameExist)
			return 0, nil
		}
	} else {
		param.Name = ""
	}
	if err := sn.Skynet.Group.Update(gid, param.Name, &param.Note); err != nil {
		return 500, err
	}

	success(logf, "Update user group")
	response(c, CodeOK)
	return 0, nil
}

func APIDeleteGroup(c *gin.Context, id uuid.UUID) (int, error) {
	var uriParam idURI
	if err := tracerr.Wrap(c.ShouldBindUri(&uriParam)); err != nil {
		return 400, err
	}
	gid, err := uuid.Parse(uriParam.ID)
	if err != nil {
		return 400, err
	}
	logf := wrap(c, id, log.Fields{
		"gid": gid,
	})

	if gid == sn.Skynet.GetID(sn.GroupRootID) {
		logf.Warn(CodeRootNotAllowedDelete.String(c))
		response(c, CodeRootNotAllowedDelete)
		return 0, nil
	}

	if err := sn.Skynet.GetDB().Transaction(func(tx *gorm.DB) error {
		_, err = sn.Skynet.Group.WithTx(tx).Delete(gid)
		if err != nil {
			return err
		}
		_, err = sn.Skynet.Permission.WithTx(tx).DeleteAll(uuid.Nil, gid)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return 500, err
	}

	success(logf, "Delete user group")
	response(c, CodeOK)
	return 0, nil
}

type userParam struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required"`
	Avatar   []byte   `json:"avatar"`
	Group    []string `json:"group" binding:"dive,uuid"`
}

func APIAddUser(c *gin.Context, id uuid.UUID) (int, error) {
	var param userParam
	var err error
	if err = tracerr.Wrap(c.ShouldBind(&param)); err != nil {
		return 400, err
	}
	if len(param.Avatar) == 0 {
		content, err := ioutil.ReadFile(viper.GetString("default_avatar"))
		if err != nil {
			return 500, err
		}
		param.Avatar = content
	}
	webp, err := utils.ConvertWebp(param.Avatar)
	if err != nil {
		return 400, err
	}
	var group []uuid.UUID
	for _, v := range param.Group {
		tmp, err := uuid.Parse(v)
		if err != nil {
			return 400, err
		}
		group = append(group, tmp)
	}
	logf := wrap(c, id, log.Fields{
		"username": param.Username,
	})

	ok, err := sn.Skynet.User.GetByName(param.Username)
	if err != nil {
		return 500, err
	}
	if ok != nil {
		logf.Warn(CodeUsernameExist.String(c))
		response(c, CodeUsernameExist)
		return 0, nil
	}

	var user *sn.User
	if err := sn.Skynet.GetDB().Transaction(func(tx *gorm.DB) error {
		for _, v := range group {
			ok, err := sn.Skynet.Group.WithTx(tx).Get(v)
			if err != nil {
				return err
			}
			if ok == nil {
				logf.Warn(CodeGroupNotExist.String(c))
				response(c, CodeGroupNotExist)
				return nil
			}
		}
		user, _, err = sn.Skynet.User.WithTx(tx).New(param.Username, param.Password, webp)
		if err != nil {
			return err
		}
		_, err = sn.Skynet.Group.WithTx(tx).Link([]uuid.UUID{user.ID}, group)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return 500, err
	}

	if user != nil {
		logf = logf.WithField("uid", user.ID)
		success(logf, "Add user")
		responseData(c, user.ID)
	}
	return 0, nil
}

func APIDeleteUser(c *gin.Context, id uuid.UUID) (int, error) {
	var uriParam idURI
	if err := tracerr.Wrap(c.ShouldBindUri(&uriParam)); err != nil {
		return 400, err
	}
	uid, err := uuid.Parse(uriParam.ID)
	if err != nil {
		return 400, err
	}
	logf := wrap(c, id, log.Fields{
		"uid": uid,
	})

	if err := sn.Skynet.GetDB().Transaction(func(tx *gorm.DB) error {
		_, err = sn.Skynet.User.WithTx(tx).Delete(uid)
		if err != nil {
			return err
		}
		_, err = sn.Skynet.Group.WithTx(tx).Unlink(uid, uuid.Nil)
		if err != nil {
			return err
		}
		_, err = sn.Skynet.Permission.WithTx(tx).DeleteAll(uid, uuid.Nil)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return 500, err
	}

	success(logf, "Delete user")
	response(c, CodeOK)
	return 0, nil
}

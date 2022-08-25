package api

import (
	"io/ioutil"
	"skynet/db"
	"skynet/handler"
	"skynet/utils/log"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func APIGetUser(req *Request) (*Response, error) {
	type Param struct {
		Text           string `form:"text"`
		LastLoginSort  string `form:"lastLoginSort" binding:"omitempty,oneof=asc desc"`
		LastLoginStart int64  `form:"lastLoginStart" binding:"min=0"`
		LastLoginEnd   int64  `form:"lastLoginEnd" binding:"min=0"`
		createdParam
		paginationParam
	}
	var param Param
	if err := req.ShouldBindQuery(&param); err != nil {
		return rspParamInvalid, err
	}

	cond := new(db.Condition)
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
	cond.MergeAnd(param.paginationParam.ToCondition())
	cond.MergeAnd(param.createdParam.ToCondition())
	cond.AndLike("id LIKE ? OR username LIKE ? OR last_ip LIKE ?", param.Text)

	rec, err := handler.User.GetAll(cond)
	if err != nil {
		return nil, err
	}
	count, err := handler.User.Count(cond)
	if err != nil {
		return nil, err
	}
	return NewPageResponse(rec, count), nil
}

func APIGetGroup(req *Request) (*Response, error) {
	type Param struct {
		Text string `form:"text"`
		createdParam
		updatedParam
		paginationParam
	}
	var param Param
	if err := req.ShouldBindQuery(&param); err != nil {
		return rspParamInvalid, err
	}

	cond := new(db.Condition)
	cond.MergeAnd(param.paginationParam.ToCondition())
	cond.MergeAnd(param.createdParam.ToCondition())
	cond.MergeAnd(param.updatedParam.ToCondition())
	cond.AndLike("id LIKE ? OR name LIKE ? OR note LIKE ?", param.Text)

	rec, err := handler.Group.GetAll(cond)
	if err != nil {
		return nil, err
	}
	count, err := handler.Group.Count(cond)
	if err != nil {
		return nil, err
	}
	return NewPageResponse(rec, count), nil
}

type groupParam struct {
	Name string `json:"name" binding:"required"`
	Note string `json:"note"`
}

func APIAddGroup(req *Request) (*Response, error) {
	type Param struct {
		groupParam
		Base      string `json:"base" binding:"omitempty,uuid"`
		CloneUser bool   `json:"clone_user"`
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return rspParamInvalid, err
	}
	var baseID uuid.UUID
	if param.Base != "" {
		baseID = uuid.MustParse(param.Base)
	}
	logger := req.Logger.WithFields(log.F{
		"name":      param.Name,
		"note":      param.Note,
		"base":      param.Base,
		"cloneUser": param.CloneUser,
	})

	var rsp *Response
	var group *db.UserGroup
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := handler.Group.WithTx(tx).GetByName(param.Name)
		if err != nil {
			return err
		}
		if ok != nil {
			logger.Warn(CodeGroupNameExist.String(req.Translator))
			rsp = &Response{Code: CodeGroupNameExist}
			return nil
		}
		group, err = handler.Group.WithTx(tx).New(param.Name, param.Note)
		if err != nil {
			return err
		}
		if baseID != uuid.Nil {
			base, err := handler.Permission.WithTx(tx).GetAll(uuid.Nil, baseID, false)
			if err != nil {
				return err
			}
			var perm []*handler.Perm
			for _, v := range base {
				perm = append(perm, &handler.Perm{
					ID:   v.PID,
					Perm: v.Perm,
				})
			}
			_, err = handler.Permission.WithTx(tx).AddToGroup(group.ID, perm)
			if err != nil {
				return err
			}
		}
		if param.CloneUser {
			user, err := handler.Group.WithTx(tx).GetGroupAllUser(baseID)
			if err != nil {
				return err
			}
			var uid []uuid.UUID
			for _, v := range user {
				uid = append(uid, v.ID)
			}
			_, err = handler.Group.WithTx(tx).Link(uid, []uuid.UUID{group.ID})
			if err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}
	if rsp != nil {
		return rsp, nil
	}
	logger = logger.WithField("gid", group.ID)
	success(logger, "Add user group")
	return &Response{Data: group.ID}, nil
}

func APIPutGroup(req *Request) (*Response, error) {
	var uriParam idURI
	if err := req.ShouldBindUri(&uriParam); err != nil {
		return rspParamInvalid, err
	}
	gid, err := uuid.Parse(uriParam.ID)
	if err != nil {
		return rspParamInvalid, err
	}
	var param groupParam
	if err := req.ShouldBind(&param); err != nil {
		return rspParamInvalid, err
	}
	logger := req.Logger.WithFields(log.F{
		"gid":  gid,
		"name": param.Name,
		"note": param.Note,
	})

	if gid == db.GetDefaultID(db.GroupRootID) && param.Name != "root" {
		logger.Warn(CodeRootNotAllowedUpdate.String(req.Translator))
		return &Response{Code: CodeRootNotAllowedUpdate}, nil
	}
	group, err := handler.Group.Get(gid)
	if err != nil {
		return nil, err
	}
	if group == nil {
		logger.Warn(CodeGroupNotExist.String(req.Translator))
		return &Response{Code: CodeGroupNotExist}, nil
	}
	if param.Name == group.Name && param.Note == group.Note {
		return rspOK, nil
	}
	if param.Name != group.Name {
		newGroup, err := handler.Group.GetByName(param.Name)
		if err != nil {
			return nil, err
		}
		if newGroup != nil {
			logger.Warn(CodeGroupNameExist.String(req.Translator))
			return &Response{Code: CodeGroupNameExist}, nil
		}
	} else {
		param.Name = ""
	}
	if err := handler.Group.Update(gid, param.Name, &param.Note); err != nil {
		return nil, err
	}

	success(logger, "Update user group")
	return rspOK, nil
}

func APIDeleteGroup(req *Request) (*Response, error) {
	var uriParam idURI
	if err := req.ShouldBindUri(&uriParam); err != nil {
		return rspParamInvalid, err
	}
	gid, err := uuid.Parse(uriParam.ID)
	if err != nil {
		return rspParamInvalid, err
	}
	logger := req.Logger.WithFields(log.F{
		"gid": gid,
	})

	if gid == db.GetDefaultID(db.GroupRootID) {
		logger.Warn(CodeRootNotAllowedDelete.String(req.Translator))
		return &Response{Code: CodeRootNotAllowedDelete}, nil
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		_, err = handler.Group.WithTx(tx).Delete(gid)
		if err != nil {
			return err
		}
		_, err = handler.Permission.WithTx(tx).DeleteAll(uuid.Nil, gid)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	success(logger, "Delete user group")
	return rspOK, nil
}

type userParam struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required"`
	Avatar   []byte   `json:"avatar"`
	Group    []string `json:"group" binding:"dive,uuid"`
}

func APIAddUser(req *Request) (*Response, error) {
	var param userParam
	var err error
	if err = req.ShouldBind(&param); err != nil {
		return rspParamInvalid, err
	}
	if len(param.Avatar) == 0 {
		content, err := ioutil.ReadFile(viper.GetString("default_avatar"))
		if err != nil {
			return nil, err
		}
		param.Avatar = content
	}
	var group []uuid.UUID
	for _, v := range param.Group {
		tmp, err := uuid.Parse(v)
		if err != nil {
			return rspParamInvalid, err
		}
		group = append(group, tmp)
	}
	logger := req.Logger.WithFields(log.F{
		"username": param.Username,
	})

	ok, err := handler.User.GetByName(param.Username)
	if err != nil {
		return nil, err
	}
	if ok != nil {
		logger.Warn(CodeUsernameExist.String(req.Translator))
		return &Response{Code: CodeUsernameExist}, nil
	}

	var user *db.User
	var rsp *Response
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		for _, v := range group {
			ok, err := handler.Group.WithTx(tx).Get(v)
			if err != nil {
				return err
			}
			if ok == nil {
				logger.Warn(CodeGroupNotExist.String(req.Translator))
				rsp = &Response{Code: CodeGroupNotExist}
				return nil
			}
		}
		user, _, err = handler.User.WithTx(tx).New(param.Username, param.Password, param.Avatar)
		if err != nil {
			return err
		}
		_, err = handler.Group.WithTx(tx).Link([]uuid.UUID{user.ID}, group)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if rsp != nil {
		return rsp, nil
	}

	logger = logger.WithField("uid", user.ID)
	success(logger, "Add user")
	return &Response{Data: user.ID}, nil
}

func APIDeleteUser(req *Request) (*Response, error) {
	var uriParam idURI
	if err := req.ShouldBindUri(&uriParam); err != nil {
		return rspParamInvalid, err
	}
	uid, err := uuid.Parse(uriParam.ID)
	if err != nil {
		return rspParamInvalid, err
	}
	logger := req.Logger.WithFields(log.F{
		"uid": uid,
	})

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		_, err = handler.User.WithTx(tx).Delete(uid)
		if err != nil {
			return err
		}
		_, err = handler.Group.WithTx(tx).Unlink(uid, uuid.Nil)
		if err != nil {
			return err
		}
		_, err = handler.Permission.WithTx(tx).DeleteAll(uid, uuid.Nil)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	success(logger, "Delete user")
	return rspOK, nil
}

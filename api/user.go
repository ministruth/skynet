package api

import (
	"os"
	"time"

	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/vincent-petithory/dataurl"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

func parseAvatar(data []*sn.User) error {
	avatar, err := os.ReadFile(viper.GetString("avatar"))
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, v := range data {
		if v.Avatar == "" {
			v.Avatar = dataurl.EncodeBytes(avatar)
		}
	}
	return nil
}

func userWrapper(req *sn.Request) (*sn.User, *sn.Response, error) {
	var uriParam sn.IDURI
	if err := req.ShouldBindUri(&uriParam); err != nil {
		return nil, sn.ResponseParamInvalid, err
	}
	uid, err := uriParam.Parse()
	if err != nil {
		return nil, sn.ResponseParamInvalid, err
	}

	user, err := sn.Skynet.User.Get(uid)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, &sn.Response{Code: sn.CodeUserNotexist}, nil
	}
	return user, nil, nil
}

func APIGetUser(req *sn.Request) (*sn.Response, error) {
	user, ret, err := userWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	return &sn.Response{Data: user}, nil
}

func APIGetUserGroup(req *sn.Request) (*sn.Response, error) {
	type Rsp struct {
		sn.GeneralFields
		Name string `json:"name"`
	}
	user, ret, err := userWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}

	link, err := sn.Skynet.Group.GetUserAllGroup(user.ID)
	if err != nil {
		return nil, err
	}
	rsp := []*Rsp{}
	for _, v := range link {
		rsp = append(rsp, &Rsp{
			GeneralFields: sn.GeneralFields{
				ID:        v.Group.ID,
				CreatedAt: v.CreatedAt,
				UpdatedAt: v.UpdatedAt,
			},
			Name: v.Group.Name,
		})
	}
	return &sn.Response{Data: rsp}, nil
}

func APIGetUsers(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Text           string `form:"text"`
		ID             string `form:"id"`
		Username       string `form:"username"`
		LastLoginSort  string `form:"lastLoginSort" binding:"omitempty,oneof=asc desc"`
		LastLoginStart int64  `form:"lastLoginStart" binding:"min=0"`
		LastLoginEnd   int64  `form:"lastLoginEnd" binding:"min=0"`
		sn.CreatedParam
		sn.PaginationParam
	}
	var param Param
	if err := req.ShouldBindQuery(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	if param.ID == "" {
		param.ID = param.Text
	}
	if param.Username == "" {
		param.Username = param.Text
	}

	cond := new(sn.Condition)
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
	cond.MergeAnd(param.PaginationParam.ToCondition())
	cond.MergeAnd(param.CreatedParam.ToCondition())
	likeCond := new(sn.Condition)
	if param.ID != "" {
		likeCond.OrLike("id LIKE ?", param.ID)
	}
	if param.Username != "" {
		likeCond.OrLike("username LIKE ?", param.Username)
	}
	if param.Text != "" {
		likeCond.OrLike("last_ip LIKE ?", param.Text)
	}
	if likeCond.Query != "" {
		cond.MergeAnd(likeCond)
	}

	rec, err := sn.Skynet.User.GetAll(cond)
	if err != nil {
		return nil, err
	}
	count, err := sn.Skynet.User.Count(cond)
	if err != nil {
		return nil, err
	}

	err = parseAvatar(rec)
	if err != nil {
		return nil, err
	}
	return sn.NewPageResponse(rec, count), nil
}

func APIAddUser(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Username   string   `json:"username" binding:"required,max=32"`
		Password   string   `json:"password" binding:"required"`
		Avatar     string   `json:"avatar"`
		Group      []string `json:"group" binding:"dive,uuid"`
		Base       string   `json:"base" binding:"omitempty,uuid"`
		CloneGroup bool     `json:"clone_group"`
	}
	var param Param
	var err error
	if err = req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	group, err := utils.ParseUUIDSlice(param.Group)
	if err != nil {
		return sn.ResponseParamInvalid, err
	}
	var baseID uuid.UUID
	if param.Base != "" {
		baseID, err = uuid.Parse(param.Base)
		if err != nil {
			return sn.ResponseParamInvalid, tracerr.Wrap(err)
		}
	}
	if baseID == uuid.Nil && param.CloneGroup {
		return sn.ResponseParamInvalid, nil
	}

	var user *sn.User
	var rsp *sn.Response
	err = sn.Skynet.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := sn.Skynet.User.WithTx(tx).GetByName(param.Username)
		if err != nil {
			return err
		}
		if ok != nil {
			rsp = &sn.Response{Code: sn.CodeUserExist}
			return nil
		}
		for _, v := range group {
			ok, err := sn.Skynet.Group.WithTx(tx).Get(v)
			if err != nil {
				return err
			}
			if ok == nil {
				rsp = &sn.Response{Code: sn.CodeGroupNotexist}
				return nil
			}
		}
		if param.Base != "" {
			ok, err := sn.Skynet.User.WithTx(tx).Get(baseID)
			if err != nil {
				return err
			}
			if ok != nil {
				rsp = &sn.Response{Code: sn.CodeUserNotexist}
				return nil
			}
		}

		user, err = sn.Skynet.User.WithTx(tx).New(param.Username, param.Password, param.Avatar)
		if err != nil {
			return err
		}
		_, err = sn.Skynet.Group.WithTx(tx).Link([]uuid.UUID{user.ID}, group)
		if err != nil {
			return err
		}
		if param.Base != "" {
			perm, err := sn.Skynet.Permission.WithTx(tx).GetUser(baseID)
			if err != nil {
				return err
			}
			_, err = sn.Skynet.Permission.WithTx(tx).AddToUser(user.ID, utils.MapValueToSlice(perm))
			if err != nil {
				return err
			}
		}
		if param.CloneGroup {
			link, err := sn.Skynet.Group.WithTx(tx).GetUserAllGroup(baseID)
			if err != nil {
				return err
			}
			var groups []uuid.UUID
			for _, v := range link {
				groups = append(groups, v.Group.ID)
			}
			_, err = sn.Skynet.Group.WithTx(tx).Link([]uuid.UUID{user.ID}, groups)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if rsp != nil {
		return rsp, nil
	}

	logger := req.Logger.WithFields(log.F{
		"username": param.Username,
		"group":    param.Group,
		"uid":      user.ID,
	})
	success(logger, "Add user")
	return &sn.Response{Data: user.ID}, nil
}

func APIPutUser(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Column   []string `json:"column" binding:"required"`
		Username string   `json:"username" binding:"max=32"`
		Password string   `json:"password"`
		Avatar   string   `json:"avatar"`
		Group    []string `json:"group" binding:"dive,uuid"`
	}
	user, ret, err := userWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	group, err := utils.ParseUUIDSlice(param.Group)
	if err != nil {
		return sn.ResponseParamInvalid, err
	}

	column := utils.SliceToMap(param.Column)
	var columns []string
	if utils.MapContains(column, "username") {
		if param.Username == "" {
			return sn.ResponseParamInvalid, nil
		}
		columns = append(columns, "username")
	}
	if utils.MapContains(column, "password") {
		if param.Password == "" {
			return sn.ResponseParamInvalid, nil
		}
		columns = append(columns, "password")
	}
	if utils.MapContains(column, "avatar") {
		columns = append(columns, "avatar")
	}
	var rsp *sn.Response
	err = sn.Skynet.DB.Transaction(func(tx *gorm.DB) error {
		if utils.MapContains(column, "username") {
			ok, err := sn.Skynet.User.WithTx(tx).GetByName(param.Username)
			if err != nil {
				return err
			}
			if ok != nil {
				rsp = &sn.Response{Code: sn.CodeUserExist}
				return nil
			}
		}
		for _, v := range group {
			ok, err := sn.Skynet.Group.WithTx(tx).Get(v)
			if err != nil {
				return err
			}
			if ok == nil {
				rsp = &sn.Response{Code: sn.CodeGroupNotexist}
				return nil
			}
		}

		if err := sn.Skynet.User.WithTx(tx).Update(columns, &sn.User{
			GeneralFields: sn.GeneralFields{ID: user.ID},
			Username:      param.Username,
			Password:      param.Password,
			Avatar:        param.Avatar,
		}); err != nil {
			return err
		}
		if err := sn.Skynet.Group.WithTx(tx).Unlink([]uuid.UUID{user.ID}, nil); err != nil {
			return err
		}
		if _, err := sn.Skynet.Group.WithTx(tx).Link([]uuid.UUID{user.ID}, group); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if rsp != nil {
		return rsp, nil
	}
	return sn.ResponseOK, nil
}

func APIDeleteUser(req *sn.Request) (*sn.Response, error) {
	user, ret, err := userWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}

	if err := sn.Skynet.User.Delete(user.ID); err != nil {
		return nil, err
	}

	logger := req.Logger.WithFields(log.F{
		"uid": user.ID,
	})
	success(logger, "Delete user")
	return sn.ResponseOK, nil
}

func APIDeleteUsers(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		ID []string `json:"id" binding:"required,dive,uuid"`
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	ids, err := utils.ParseUUIDSlice(param.ID)
	if err != nil {
		return sn.ResponseParamInvalid, err
	}

	err = sn.Skynet.DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if err := sn.Skynet.User.WithTx(tx).Delete(id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	logger := req.Logger.WithFields(log.F{
		"uid": ids,
	})
	success(logger, "Delete users")
	return sn.ResponseOK, nil
}

func APIKickUser(req *sn.Request) (*sn.Response, error) {
	user, ret, err := userWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}

	if err := sn.Skynet.User.Kick(user.ID); err != nil {
		return nil, err
	}
	return sn.ResponseOK, nil
}

func APIGetUserPermission(req *sn.Request) (*sn.Response, error) {
	type Rsp struct {
		sn.GeneralFields
		Name   string      `json:"name"`
		Note   string      `json:"note"`
		Perm   sn.UserPerm `json:"perm"`
		Origin []*Rsp      `json:"origin,omitempty"`
	}
	user, ret, err := userWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}

	perm, err := sn.Skynet.Permission.GetUserMerged(user.ID)
	if err != nil {
		return nil, err
	}
	data := []*Rsp{}
	for _, v := range perm {
		var origin []*Rsp
		if v.Origin != nil {
			for _, v := range v.Origin {
				origin = append(origin, &Rsp{
					GeneralFields: sn.GeneralFields{
						ID:        v.ID,
						CreatedAt: v.CreatedAt,
						UpdatedAt: v.UpdatedAt,
					},
					Name:   v.Name,
					Note:   v.Note,
					Perm:   v.Perm,
					Origin: nil,
				})
			}
		}
		data = append(data, &Rsp{
			GeneralFields: sn.GeneralFields{
				ID:        v.ID,
				CreatedAt: v.CreatedAt,
				UpdatedAt: v.UpdatedAt,
			},
			Name:   v.Name,
			Note:   v.Note,
			Perm:   v.Perm,
			Origin: origin,
		})
	}
	return &sn.Response{Data: data}, nil
}

func APIPutUserPermission(req *sn.Request) (*sn.Response, error) {
	type ParsedParam struct {
		ID   uuid.UUID
		Perm sn.UserPerm
	}
	type Param struct {
		ID   string      `json:"id" binding:"required,uuid"`
		Perm sn.UserPerm `json:"perm" binding:"min=-1,max=7"`
	}
	user, ret, err := userWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	var param []Param
	if err := req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	if len(param) == 0 {
		return sn.ResponseOK, nil
	}

	perm, err := sn.Skynet.Permission.GetEntry()
	if err != nil {
		return nil, err
	}
	data := make(map[uuid.UUID]bool)
	for _, v := range perm {
		data[v.ID] = true
	}
	var params []*ParsedParam
	for _, v := range param {
		id, err := uuid.Parse(v.ID)
		if err != nil {
			return sn.ResponseParamInvalid, err
		}
		if !utils.MapContains(data, id) {
			return &sn.Response{Code: sn.CodePermissionNotexist}, nil
		}
		params = append(params, &ParsedParam{
			ID:   id,
			Perm: v.Perm,
		})
	}
	err = sn.Skynet.DB.Transaction(func(tx *gorm.DB) error {
		impl := sn.Skynet.Permission.WithTx(tx)
		for _, v := range params {
			err = impl.Grant(user.ID, uuid.Nil, v.ID, v.Perm)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sn.ResponseOK, nil
}

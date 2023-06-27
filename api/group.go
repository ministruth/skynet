package api

import (
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

func groupWrapper(req *sn.Request) (*sn.Group, *sn.Response, error) {
	var uriParam sn.IDURI
	if err := req.ShouldBindUri(&uriParam); err != nil {
		return nil, sn.ResponseParamInvalid, err
	}
	gid, err := uriParam.Parse()
	if err != nil {
		return nil, sn.ResponseParamInvalid, err
	}

	group, err := sn.Skynet.Group.Get(gid)
	if err != nil {
		return nil, nil, err
	}
	if group == nil {
		return nil, &sn.Response{Code: sn.CodeGroupNotexist}, nil
	}
	return group, nil, nil
}

func APIGetGroup(req *sn.Request) (*sn.Response, error) {
	group, ret, err := groupWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	return &sn.Response{Data: group}, nil
}

func APIGetGroupUser(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Text string `form:"text"`
		sn.PaginationParam
	}
	type Rsp struct {
		sn.GeneralFields
		Username string `json:"username"`
	}

	group, ret, err := groupWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	var param Param
	if err := req.ShouldBindQuery(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}

	cond := new(sn.Condition)
	cond.MergeAnd(param.PaginationParam.ToCondition())
	if param.Text != "" {
		cond.AndLike("User.id LIKE ? OR User.username LIKE ?", param.Text)
	}
	link, err := sn.Skynet.Group.GetGroupAllUser(group.ID, cond)
	if err != nil {
		return nil, err
	}
	count, err := sn.Skynet.Group.CountGroupAllUser(group.ID, nil)
	if err != nil {
		return nil, err
	}

	rsp := []*Rsp{}
	for _, v := range link {
		rsp = append(rsp, &Rsp{
			GeneralFields: sn.GeneralFields{
				ID:        v.User.ID,
				CreatedAt: v.CreatedAt,
				UpdatedAt: v.UpdatedAt,
			},
			Username: v.User.Username,
		})
	}
	return sn.NewPageResponse(rsp, count), nil
}

func APIGetGroups(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Text string `form:"text"`
		ID   string `form:"id"`
		Name string `form:"name"`
		sn.CreatedParam
		sn.UpdatedParam
		sn.PaginationParam
	}
	var param Param
	if err := req.ShouldBindQuery(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	if param.ID == "" {
		param.ID = param.Text
	}
	if param.Name == "" {
		param.Name = param.Text
	}

	cond := new(sn.Condition)
	cond.MergeAnd(param.PaginationParam.ToCondition())
	cond.MergeAnd(param.CreatedParam.ToCondition())
	cond.MergeAnd(param.UpdatedParam.ToCondition())
	likeCond := new(sn.Condition)
	if param.ID != "" {
		likeCond.OrLike("id LIKE ?", param.ID)
	}
	if param.Name != "" {
		likeCond.OrLike("name LIKE ?", param.Name)
	}
	if param.Text != "" {
		likeCond.OrLike("note LIKE ?", param.Text)
	}
	if likeCond.Query != "" {
		cond.MergeAnd(likeCond)
	}

	rec, err := sn.Skynet.Group.GetAll(cond)
	if err != nil {
		return nil, err
	}
	count, err := sn.Skynet.Group.Count(cond)
	if err != nil {
		return nil, err
	}

	return sn.NewPageResponse(rec, count), nil
}

func APIAddGroup(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Name      string `json:"name" binding:"required,max=32"`
		Note      string `json:"note" binding:"max=256"`
		Base      string `json:"base" binding:"omitempty,uuid"`
		CloneUser bool   `json:"clone_user"`
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	var baseID uuid.UUID
	var err error
	if param.Base != "" {
		baseID, err = uuid.Parse(param.Base)
		if err != nil {
			return sn.ResponseParamInvalid, tracerr.Wrap(err)
		}
	}
	if baseID == uuid.Nil && param.CloneUser {
		return sn.ResponseParamInvalid, nil
	}

	var rsp *sn.Response
	var group *sn.Group
	err = sn.Skynet.DB.Transaction(func(tx *gorm.DB) error {
		// check param
		ok, err := sn.Skynet.Group.WithTx(tx).GetByName(param.Name)
		if err != nil {
			return err
		}
		if ok != nil {
			rsp = &sn.Response{Code: sn.CodeGroupExist}
			return nil
		}
		if baseID != uuid.Nil {
			ok, err := sn.Skynet.Group.WithTx(tx).Get(baseID)
			if err != nil {
				return err
			}
			if ok == nil {
				rsp = &sn.Response{Code: sn.CodeGroupNotexist}
				return nil
			}
		}

		group, err = sn.Skynet.Group.WithTx(tx).New(param.Name, param.Note)
		if err != nil {
			return err
		}
		if baseID != uuid.Nil {
			base, err := sn.Skynet.Permission.WithTx(tx).GetAll(uuid.Nil, baseID, false, false, false)
			if err != nil {
				return err
			}
			var perm []*sn.PermEntry
			for _, v := range base {
				perm = append(perm, &sn.PermEntry{
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
			link, err := sn.Skynet.Group.WithTx(tx).GetGroupAllUser(baseID, nil)
			if err != nil {
				return err
			}
			var uid []uuid.UUID
			for _, v := range link {
				uid = append(uid, v.User.ID)
			}
			_, err = sn.Skynet.Group.WithTx(tx).Link(uid, []uuid.UUID{group.ID})
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
	return &sn.Response{Data: group.ID}, nil
}

func APIPutGroup(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		Column []string `json:"column" binding:"required"`
		Name   string   `json:"name" binding:"max=32"`
		Note   string   `json:"note" binding:"max=256"`
	}
	group, ret, err := groupWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}

	column := utils.SliceToMap(param.Column)
	var columns []string
	if group.ID == sn.Skynet.ID.Get(sn.GroupRootID) && utils.MapContains(column, "name") && param.Name != "root" {
		return &sn.Response{Code: sn.CodeGroupRootupdate}, nil
	}
	if utils.MapContains(column, "name") {
		if param.Name == "" {
			return sn.ResponseParamInvalid, nil
		}
		ok, err := sn.Skynet.Group.GetByName(param.Name)
		if err != nil {
			return nil, err
		}
		if ok != nil {
			return &sn.Response{Code: sn.CodeGroupExist}, nil
		}
		columns = append(columns, "name")
	}
	if utils.MapContains(column, "note") {
		columns = append(columns, "note")
	}

	if err := sn.Skynet.Group.Update(columns, &sn.Group{
		GeneralFields: sn.GeneralFields{ID: group.ID},
		Name:          param.Name,
		Note:          param.Note}); err != nil {
		return nil, err
	}
	return sn.ResponseOK, nil
}

func APIDeleteGroup(req *sn.Request) (*sn.Response, error) {
	group, ret, err := groupWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}

	if group.ID == sn.Skynet.ID.Get(sn.GroupRootID) {
		return &sn.Response{Code: sn.CodeGroupRootdelete}, nil
	}

	if err := sn.Skynet.Group.Delete(group.ID); err != nil {
		return nil, err
	}

	logger := req.Logger.WithFields(log.F{
		"gid": group.ID,
	})
	success(logger, "Delete group")
	return sn.ResponseOK, nil
}

func APIDeleteGroups(req *sn.Request) (*sn.Response, error) {
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
	for _, id := range ids {
		if id == sn.Skynet.ID.Get(sn.GroupRootID) {
			return &sn.Response{Code: sn.CodeGroupRootdelete}, nil
		}
	}

	err = sn.Skynet.DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			err := sn.Skynet.Group.WithTx(tx).Delete(id)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	logger := req.Logger.WithFields(log.F{
		"gid": ids,
	})
	success(logger, "Delete groups")
	return sn.ResponseOK, nil
}

func APIDeleteGroupUsers(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		ID []string `json:"id" binding:"required,dive,uuid"`
	}
	group, ret, err := groupWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	ids, err := utils.ParseUUIDSlice(param.ID)
	if err != nil {
		return sn.ResponseParamInvalid, err
	}

	if err := sn.Skynet.Group.Unlink(ids, []uuid.UUID{group.ID}); err != nil {
		return nil, err
	}
	logger := req.Logger.WithFields(log.F{
		"gid": group.ID,
		"uid": param.ID,
	})
	success(logger, "Delete group users")
	return sn.ResponseOK, nil
}

func APIAddGroupUsers(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		ID []string `json:"id" binding:"required,dive,uuid"`
	}
	group, ret, err := groupWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	var param Param
	if err := req.ShouldBind(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	ids, err := utils.ParseUUIDSlice(param.ID)
	if err != nil {
		return sn.ResponseParamInvalid, err
	}

	var rsp *sn.Response
	err = sn.Skynet.DB.Transaction(func(tx *gorm.DB) error {
		for _, v := range ids {
			u, err := sn.Skynet.User.WithTx(tx).Get(v)
			if err != nil {
				return err
			}
			if u == nil {
				rsp = &sn.Response{Code: sn.CodeUserNotexist}
				return nil
			}
		}

		link, err := sn.Skynet.Group.WithTx(tx).GetGroupAllUser(group.ID, nil)
		if err != nil {
			return err
		}
		var uid []uuid.UUID
		for _, v := range link {
			uid = append(uid, v.User.ID)
		}
		linkid := utils.SliceRemove(ids, uid)
		if len(linkid) > 0 {
			if _, err := sn.Skynet.Group.WithTx(tx).Link(linkid, []uuid.UUID{group.ID}); err != nil {
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

	return sn.ResponseOK, nil
}

func APIDeleteGroupUser(req *sn.Request) (*sn.Response, error) {
	type Param struct {
		ID string `uri:"uid" binding:"required,uuid"`
	}
	group, ret, err := groupWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	var param Param
	if err := req.ShouldBindUri(&param); err != nil {
		return sn.ResponseParamInvalid, err
	}
	uid, err := uuid.Parse(param.ID)
	if err != nil {
		return sn.ResponseParamInvalid, tracerr.Wrap(err)
	}

	u, err := sn.Skynet.User.Get(uid)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return &sn.Response{Code: sn.CodeUserNotexist}, nil
	}

	if err := sn.Skynet.Group.Unlink([]uuid.UUID{uid}, []uuid.UUID{group.ID}); err != nil {
		return nil, err
	}
	logger := req.Logger.WithFields(log.F{
		"gid": group.ID,
		"uid": param.ID,
	})
	success(logger, "Delete group user")
	return sn.ResponseOK, nil
}

func APIGetGroupPermission(req *sn.Request) (*sn.Response, error) {
	type Rsp struct {
		sn.GeneralFields
		Name string      `json:"name"`
		Note string      `json:"note"`
		Perm sn.UserPerm `json:"perm"`
	}
	group, ret, err := groupWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}

	perm, err := sn.Skynet.Permission.GetAll(uuid.Nil, group.ID, false, false, true)
	if err != nil {
		return nil, err
	}
	data := []*Rsp{}
	for _, v := range perm {
		data = append(data, &Rsp{
			GeneralFields: sn.GeneralFields{
				ID:        v.PID,
				CreatedAt: v.CreatedAt,
				UpdatedAt: v.UpdatedAt,
			},
			Name: v.Permission.Name,
			Note: v.Permission.Note,
			Perm: v.Perm,
		})
	}
	return &sn.Response{Data: data}, nil
}

func APIPutGroupPermission(req *sn.Request) (*sn.Response, error) {
	type ParsedParam struct {
		ID   uuid.UUID
		Perm sn.UserPerm
	}
	type Param struct {
		ID   string      `json:"id" binding:"required,uuid"`
		Perm sn.UserPerm `json:"perm" binding:"min=-1,max=7"`
	}
	group, ret, err := groupWrapper(req)
	if ret != nil || err != nil {
		return ret, err
	}
	if group.ID == sn.Skynet.ID.Get(sn.GroupRootID) {
		return &sn.Response{Code: sn.CodeGroupRootupdate}, nil
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
			err = impl.Grant(uuid.Nil, group.ID, v.ID, v.Perm)
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

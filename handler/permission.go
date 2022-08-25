package handler

import (
	"skynet/db"
	"skynet/utils/tpl"

	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

var ErrUUIDConflict = tracerr.New("uuid conflict")

type Perm struct {
	ID   uuid.UUID
	Name string // filled automatically
	Perm db.UserPerm
}

type PermListType int32

const (
	PermListUser PermListType = iota
	PermListGroup
)

type PermList struct {
	Perm  tpl.SafeMap[uuid.UUID, *Perm]
	Group []uuid.UUID
	Type  PermListType
}

type PermissionImpl struct {
	orm   *db.ORM[db.Permission]
	cache *tpl.SafeMap[uuid.UUID, *PermList]
}

var Permission = &PermissionImpl{
	cache: new(tpl.SafeMap[uuid.UUID, *PermList]),
}

func (p *PermissionImpl) WithTx(tx *gorm.DB) *PermissionImpl {
	return &PermissionImpl{
		orm:   db.NewORM[db.Permission](tx),
		cache: p.cache,
	}
}

// AddToGroup add permission to group gid.
func (p *PermissionImpl) AddToGroup(gid uuid.UUID, perm []*Perm) (permRet []*db.Permission, err error) {
	if len(perm) == 0 {
		return nil, nil
	}
	for _, v := range perm {
		permRet = append(permRet, &db.Permission{
			GID:  gid,
			PID:  v.ID,
			Perm: v.Perm,
		})
	}
	err = p.orm.Creates(permRet)
	if err != nil {
		p.cache.Delete(gid)
	}
	return
}

func (p *PermissionImpl) newPermList(perm []*db.Permission, t PermListType) *PermList {
	ret := new(PermList)
	for _, v := range perm {
		perm := &Perm{
			ID:   v.PID,
			Perm: v.Perm,
		}
		if v.PermissionList != nil {
			perm.Name = v.PermissionList.Name
		}
		ret.Perm.Set(v.PID, perm)
	}
	ret.Type = t
	return ret
}

// GetUserMerged returns merged user permission list.
func (p *PermissionImpl) GetUserMerged(id uuid.UUID) (map[uuid.UUID]*Perm, error) {
	if id == uuid.Nil {
		return make(map[uuid.UUID]*Perm), nil
	}
	userPerm, err := p.GetUser(id, false)
	if err != nil {
		return nil, err
	}
	ret := userPerm.Perm.Map()
	for _, v := range userPerm.Group {
		gp, err := p.GetGroup(v, false)
		if err != nil {
			return nil, err
		}
		gp.Perm.Range(func(k uuid.UUID, v *Perm) bool {
			if _, ok := ret[k]; !ok { // not allow override
				ret[k] = v
			}
			return true
		})
	}
	return ret, nil
}

// GetUser get user perm list, if force ignore cache.
func (p *PermissionImpl) GetUser(id uuid.UUID, force bool) (ret *PermList, err error) {
	if !force {
		if v, ok := p.cache.Get(id); ok {
			if v.Type != PermListUser {
				return nil, ErrUUIDConflict
			}
			return v, nil
		}
	}

	err = p.orm.TX().Transaction(func(tx *gorm.DB) error {
		// user perm
		userPerm, err := p.WithTx(tx).GetAll(id, uuid.Nil, true)
		if err != nil {
			return err
		}
		// group perm
		groupPerm, err := Group.WithTx(tx).GetUserAllGroup(id)
		if err != nil {
			return err
		}

		ret = p.newPermList(userPerm, PermListUser)
		for _, v := range groupPerm {
			// trigger cache init
			if _, err := p.WithTx(tx).GetGroup(v.ID, false); err != nil {
				return err
			}
			ret.Group = append(ret.Group, v.ID)
		}
		p.cache.Set(id, ret)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return
}

// GetGroup get group perm list, if force ignore cache.
func (p *PermissionImpl) GetGroup(id uuid.UUID, force bool) (*PermList, error) {
	if !force {
		if v, ok := p.cache.Get(id); ok {
			if v.Type != PermListGroup {
				return nil, ErrUUIDConflict
			}
			return v, nil
		}
	}
	res, err := p.GetAll(uuid.Nil, id, true)
	if err != nil {
		return nil, err
	}
	ret := p.newPermList(res, PermListGroup)
	p.cache.Set(id, ret)
	return ret, nil
}

// GetAll find all permission by condition, if join is true,
// return records will join PermissionList.
//
// uid and gid should not be both uuid.Nil or both have value, otherwise,
// nil,nil will be returned
func (p *PermissionImpl) GetAll(uid uuid.UUID, gid uuid.UUID, join bool) ([]*db.Permission, error) {
	if (uid != uuid.Nil && gid != uuid.Nil) || (uid == uuid.Nil && gid == uuid.Nil) {
		return nil, nil
	}

	orm := p.orm
	if uid != uuid.Nil {
		orm = orm.Where("permissions.uid = ?", uid)
	} else {
		orm = orm.Where("permissions.gid = ?", gid)
	}
	if join {
		orm = orm.Joins("PermissionList")
	}

	return orm.Find()
}

// DeleteAll delete all uid or gid permission.
//
// Note: uid or gid is uuid.Nil means not base on this condition.
// If both uuid.Nil, do nothing. If both given, delete using OR condition.
func (p *PermissionImpl) DeleteAll(uid uuid.UUID, gid uuid.UUID) (row int64, err error) {
	if uid == uuid.Nil && gid == uuid.Nil {
		return 0, nil
	}
	err = p.orm.TX().Transaction(func(tx *gorm.DB) error {
		tp := p.WithTx(tx)
		orm := tp.orm
		if uid != uuid.Nil {
			if gid != uuid.Nil {
				orm = orm.Where("uid = ? OR gid = ?", uid, gid)
			} else {
				orm = orm.Where("uid = ?", uid)
			}
		} else {
			orm = orm.Where("gid = ?", gid)
		}
		row, err = orm.Delete()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if uid != uuid.Nil {
		p.cache.Delete(uid)
	}
	if gid != uuid.Nil {
		p.cache.Delete(gid)
	}
	return
}

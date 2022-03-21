package impl

import (
	"skynet/sn"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

var ErrUUIDConflict = tracerr.New("uuid conflict")

func newPermList(p []*sn.Permission, t sn.SNPermListType) *sn.SNPermList {
	tmp := new(sn.SNPermList)
	for _, v := range p {
		perm := sn.SNPerm{
			ID:   v.PID,
			Perm: v.Perm,
		}
		if v.PermissionList != nil {
			perm.Name = v.PermissionList.Name
		}
		tmp.Perm.Set(v.PID, &perm)
	}
	tmp.Type = t
	return tmp
}

func CheckPerm(perm map[uuid.UUID]*sn.SNPerm, target *sn.SNPerm) bool {
	if perm != nil {
		if p, ok := perm[target.ID]; ok { // user permission check first
			return (p.Perm & target.Perm) == target.Perm
		}
		if p, ok := perm[sn.Skynet.GetID(sn.PermAllID)]; ok {
			return (p.Perm & target.Perm) == target.Perm
		}
	}
	if target.ID == sn.Skynet.GetID(sn.PermUserID) || target.ID == sn.Skynet.GetID(sn.PermGuestID) {
		return true
	}
	return false // fail safe
}

// GetPerm returns merged permission list.
//
// Warning: user or group id is not checked.
func GetPerm(id uuid.UUID) (map[uuid.UUID]*sn.SNPerm, error) {
	if id == uuid.Nil {
		return make(map[uuid.UUID]*sn.SNPerm), nil
	}
	var err error
	ret := make(map[uuid.UUID]*sn.SNPerm)
	p, ok := sn.Skynet.PermList.Get(id)
	if !ok {
		p, err = RefreshUserPerm(id, true)
		if err != nil {
			return nil, err
		}
	}
	if p.Type == sn.PermListGroup {
		return p.Perm.Map(), nil
	} else {
		ret = p.Perm.Map()
		for _, v := range p.Group {
			gp, ok := sn.Skynet.PermList.Get(v)
			if !ok {
				if gp, err = RefreshGroupPerm(nil, v, true); err != nil {
					return nil, err
				}
			}
			if gp.Type != sn.PermListGroup {
				log.Error("Inconsistency meet, maybe trigger a bug")
				continue
			}
			gp.Perm.Range(func(k uuid.UUID, v *sn.SNPerm) bool {
				if _, ok := ret[k]; !ok { // not allow override
					ret[k] = v
				}
				return true
			})
		}
		return ret, nil
	}
}

// RefreshUserPerm refresh user perm list, if force ignore cache.
//
// Warning: user id is not checked.
func RefreshUserPerm(id uuid.UUID, force bool) (ret *sn.SNPermList, err error) {
	if !force {
		if v, ok := sn.Skynet.PermList.Get(id); ok {
			if v.Type != sn.PermListUser {
				return nil, ErrUUIDConflict
			}
			return v, nil
		}
	}

	err = sn.Skynet.GetDB().Transaction(func(tx *gorm.DB) error {
		// user perm
		p, err := sn.Skynet.Permission.WithTx(tx).GetAll(id, uuid.Nil, true)
		if err != nil {
			return err
		}
		// group perm
		group, err := sn.Skynet.Group.WithTx(tx).GetUserAllGroup(id)
		if err != nil {
			return err
		}

		ret = newPermList(p, sn.PermListUser)
		for _, v := range group {
			if _, err := RefreshGroupPerm(tx, v.ID, false); err != nil {
				return err
			}
			ret.Group = append(ret.Group, v.ID)
		}
		sn.Skynet.PermList.Set(id, ret)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return
}

// RefreshGroupPerm refresh group perm list, if force ignore cache.
//
// Warning: group id is not checked.
func RefreshGroupPerm(tx *gorm.DB, id uuid.UUID, force bool) (*sn.SNPermList, error) {
	if !force {
		if v, ok := sn.Skynet.PermList.Get(id); ok {
			if v.Type != sn.PermListGroup {
				return nil, ErrUUIDConflict
			}
			return v, nil
		}
	}
	if tx == nil {
		tx = sn.Skynet.GetDB()
	}
	p, err := sn.Skynet.Permission.WithTx(tx).GetAll(uuid.Nil, id, true)
	if err != nil {
		return nil, err
	}
	ret := newPermList(p, sn.PermListGroup)
	sn.Skynet.PermList.Set(id, ret)
	return ret, nil
}

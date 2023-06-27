package handler

import (
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermissionImpl struct {
	tx   *gorm.DB
	link *sn.ORM[sn.PermissionLink]
	perm *sn.ORM[sn.Permission]
}

func NewPermissionHandler() sn.PermissionHandler {
	return &PermissionImpl{
		tx:   sn.Skynet.DB,
		link: sn.NewORM[sn.PermissionLink](sn.Skynet.DB),
		perm: sn.NewORM[sn.Permission](sn.Skynet.DB),
	}
}

func (impl *PermissionImpl) WithTx(tx *gorm.DB) sn.PermissionHandler {
	return &PermissionImpl{
		tx:   tx,
		link: sn.NewORM[sn.PermissionLink](tx),
		perm: sn.NewORM[sn.Permission](tx),
	}
}

func (impl *PermissionImpl) Grant(uid uuid.UUID, gid uuid.UUID, pid uuid.UUID, perm sn.UserPerm) error {
	if uid == uuid.Nil && gid == uuid.Nil {
		return nil
	}
	return impl.tx.Transaction(func(tx *gorm.DB) error {
		impl := impl.WithTx(tx)
		if uid != uuid.Nil {
			_, err := impl.DeleteUser(uid, pid)
			if err != nil {
				return err
			}
			if perm != -1 {
				impl.AddToUser(uid, []*sn.PermEntry{{
					ID:   pid,
					Perm: perm,
				}})
			}
		}
		if gid != uuid.Nil {
			_, err := impl.DeleteGroup(gid, pid)
			if err != nil {
				return err
			}
			if perm != -1 {
				impl.AddToGroup(gid, []*sn.PermEntry{{
					ID:   pid,
					Perm: perm,
				}})
			}
		}
		return nil
	})
}

func (impl *PermissionImpl) AddToUser(uid uuid.UUID, perm []*sn.PermEntry) (ret []*sn.PermissionLink, err error) {
	if len(perm) == 0 {
		return nil, nil
	}
	for _, v := range perm {
		ret = append(ret, &sn.PermissionLink{
			UID:  uuid.NullUUID{UUID: uid, Valid: true},
			PID:  v.ID,
			Perm: v.Perm,
		})
	}
	err = impl.link.Creates(ret)
	return
}

func (impl *PermissionImpl) AddToGroup(gid uuid.UUID, perm []*sn.PermEntry) (ret []*sn.PermissionLink, err error) {
	if len(perm) == 0 {
		return nil, nil
	}
	for _, v := range perm {
		ret = append(ret, &sn.PermissionLink{
			GID:  uuid.NullUUID{UUID: gid, Valid: true},
			PID:  v.ID,
			Perm: v.Perm,
		})
	}
	err = impl.link.Creates(ret)
	return
}

func (impl *PermissionImpl) toPermEntry(perm []*sn.PermissionLink) map[uuid.UUID]*sn.PermEntry {
	ret := make(map[uuid.UUID]*sn.PermEntry)
	for _, v := range perm {
		var origin []*sn.PermEntry
		if v.Group != nil {
			origin = []*sn.PermEntry{{
				ID:        v.Group.ID,
				Name:      v.Group.Name,
				Note:      v.Group.Note,
				Perm:      v.Perm,
				Origin:    nil,
				CreatedAt: v.CreatedAt,
				UpdatedAt: v.UpdatedAt,
			}}
		}
		entry := &sn.PermEntry{
			ID:        v.PID,
			Perm:      v.Perm,
			Origin:    origin,
			CreatedAt: v.CreatedAt,
			UpdatedAt: v.UpdatedAt,
		}
		if v.Permission != nil {
			entry.Name = v.Permission.Name
			entry.Note = v.Permission.Note
		}
		ret[entry.ID] = entry
	}
	return ret
}

func (impl *PermissionImpl) GetUserMerged(uid uuid.UUID) (ret map[uuid.UUID]*sn.PermEntry, err error) {
	if uid == uuid.Nil {
		return make(map[uuid.UUID]*sn.PermEntry), nil
	}
	err = impl.tx.Transaction(func(tx *gorm.DB) error {
		impl := impl.WithTx(tx)
		ret = make(map[uuid.UUID]*sn.PermEntry)
		link, err := sn.Skynet.Group.WithTx(tx).GetUserAllGroup(uid)
		if err != nil {
			return err
		}
		for _, v := range link {
			groupPerm, err := impl.GetGroup(v.Group.ID)
			if err != nil {
				return err
			}
			for k, v := range groupPerm {
				if e, ok := ret[k]; ok {
					e.Perm = e.Perm | v.Perm
					e.CreatedAt = utils.Min(e.CreatedAt, v.CreatedAt)
					e.UpdatedAt = utils.Max(e.UpdatedAt, v.UpdatedAt)
					e.Origin = append(e.Origin, v.Origin...)
				} else {
					ret[k] = v
				}
			}
		}
		userPerm, err := impl.GetUser(uid)
		if err != nil {
			return err
		}
		for k, v := range userPerm {
			ret[k] = v
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return
}

func (impl *PermissionImpl) GetUser(uid uuid.UUID) (map[uuid.UUID]*sn.PermEntry, error) {
	res, err := impl.GetAll(uid, uuid.Nil, false, false, true)
	if err != nil {
		return nil, err
	}
	ret := impl.toPermEntry(res)
	return ret, nil
}

func (impl *PermissionImpl) GetEntry() ([]*sn.Permission, error) {
	return impl.perm.Find()
}

func (impl *PermissionImpl) GetGroup(gid uuid.UUID) (map[uuid.UUID]*sn.PermEntry, error) {
	res, err := impl.GetAll(uuid.Nil, gid, false, true, true)
	if err != nil {
		return nil, err
	}
	ret := impl.toPermEntry(res)
	return ret, nil
}

func (impl *PermissionImpl) GetAll(uid uuid.UUID, gid uuid.UUID,
	joinUser bool, joinGroup bool, joinPerm bool) ([]*sn.PermissionLink, error) {
	if (uid != uuid.Nil && gid != uuid.Nil) || (uid == uuid.Nil && gid == uuid.Nil) {
		return nil, nil
	}

	link := impl.link
	if uid != uuid.Nil {
		link = link.Where("permission_links.uid = ?", uid)
	} else {
		link = link.Where("permission_links.gid = ?", gid)
	}
	if joinUser {
		link = link.Joins("User")
	}
	if joinGroup {
		link = link.Joins("Group")
	}
	if joinPerm {
		link = link.Joins("Permission")
	}

	return link.Find()
}

func (impl *PermissionImpl) Delete(id uuid.UUID) error {
	_, err := impl.link.DeleteID(id)
	return err
}

func (impl *PermissionImpl) DeleteUser(uid uuid.UUID, pid uuid.UUID) (int64, error) {
	if pid == uuid.Nil {
		return impl.link.Delete("uid = ?", uid)
	} else {
		return impl.link.Delete("uid = ? AND pid = ?", uid, pid)
	}
}

func (impl *PermissionImpl) DeleteGroup(gid uuid.UUID, pid uuid.UUID) (int64, error) {
	if pid == uuid.Nil {
		return impl.link.Delete("gid = ?", gid)
	} else {
		return impl.link.Delete("gid = ? AND pid = ?", gid, pid)
	}
}

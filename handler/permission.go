package handler

import (
	"skynet/sn"
	"skynet/sn/impl"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type sitePermission struct {
	*impl.ORM[sn.Permission]
}

func NewPermission() sn.SNPermission {
	return &sitePermission{
		ORM: impl.NewORM[sn.Permission](nil),
	}
}

func (p *sitePermission) WithTx(tx *gorm.DB) sn.SNPermission {
	return &sitePermission{
		ORM: impl.NewORM[sn.Permission](tx),
	}
}

func (p *sitePermission) DeleteAll(uid uuid.UUID, gid uuid.UUID) (row int64, err error) {
	if uid == uuid.Nil && gid == uuid.Nil {
		return 0, nil
	}
	err = p.TX().Transaction(func(tx *gorm.DB) error {
		tp := p.WithTx(tx).(*sitePermission)
		imp := tp.Impl
		if uid != uuid.Nil {
			if gid != uuid.Nil {
				imp = imp.Where("uid = ? OR gid = ?", uid, gid)
			} else {
				imp = imp.Where("uid = ?", uid)
			}
		} else {
			imp = imp.Where("gid = ?", gid)
		}
		row, err = imp.Delete()
		if err != nil {
			return err
		}
		if uid != uuid.Nil {
			sn.Skynet.PermList.Delete(uid)
		}
		if gid != uuid.Nil {
			sn.Skynet.PermList.Delete(gid)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return
}

func (p *sitePermission) GetAll(uid uuid.UUID, gid uuid.UUID, join bool) ([]*sn.Permission, error) {
	if (uid != uuid.Nil && gid != uuid.Nil) || (uid == uuid.Nil && gid == uuid.Nil) {
		return nil, nil
	}

	imp := p.Impl
	if uid != uuid.Nil {
		imp = imp.Where("permissions.uid = ?", uid)
	} else {
		imp = imp.Where("permissions.gid = ?", gid)
	}
	if join {
		imp = imp.Joins("PermissionList")
	}

	return imp.Find()
}

func (p *sitePermission) AddToGroup(gid uuid.UUID, perm []*sn.SNPerm) (permRet []*sn.Permission, err error) {
	if len(perm) == 0 {
		return nil, nil
	}
	for _, v := range perm {
		permRet = append(permRet, &sn.Permission{
			GID:  gid,
			PID:  v.ID,
			Perm: v.Perm,
		})
	}
	err = p.Impl.Creates(permRet)
	return
}

package handler

import (
	"github.com/MXWXZ/skynet/sn"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupImpl struct {
	tx    *gorm.DB
	group *sn.ORM[sn.Group]
	link  *sn.ORM[sn.UserGroupLink]
}

func NewGroupHandler() sn.GroupHandler {
	return &GroupImpl{
		tx:    sn.Skynet.DB,
		group: sn.NewORM[sn.Group](sn.Skynet.DB),
		link:  sn.NewORM[sn.UserGroupLink](sn.Skynet.DB),
	}
}

func (u *GroupImpl) WithTx(tx *gorm.DB) sn.GroupHandler {
	return &GroupImpl{
		tx:    tx,
		group: sn.NewORM[sn.Group](tx),
		link:  sn.NewORM[sn.UserGroupLink](tx),
	}
}

func (u *GroupImpl) New(name string, note string) (*sn.Group, error) {
	group := &sn.Group{
		Name: name,
		Note: note,
	}
	if err := u.group.Create(group); err != nil {
		return nil, err
	}
	return group, nil
}

func (u *GroupImpl) Link(uid []uuid.UUID, gid []uuid.UUID) ([]*sn.UserGroupLink, error) {
	if len(uid) == 0 || len(gid) == 0 {
		return nil, nil
	}
	var group []*sn.UserGroupLink
	for _, v := range uid {
		for _, g := range gid {
			group = append(group, &sn.UserGroupLink{
				UID: v,
				GID: g,
			})
		}
	}
	if err := u.link.Creates(group); err != nil {
		return nil, err
	}
	return group, nil
}

func (impl *GroupImpl) GetGroupAllUser(id uuid.UUID, cond *sn.Condition) ([]*sn.UserGroupLink, error) {
	link, err := impl.link.Where("user_group_links.gid = ?", id).Cond(cond).Joins("User").Find()
	if err != nil {
		return nil, err
	}
	return link, nil
}

func (impl *GroupImpl) CountGroupAllUser(id uuid.UUID, cond *sn.Condition) (int64, error) {
	return impl.link.Where("user_group_links.gid = ?", id).Count(cond)
}

func (impl *GroupImpl) GetAll(cond *sn.Condition) ([]*sn.Group, error) {
	return impl.group.Cond(cond).Find()
}

func (impl *GroupImpl) Get(id uuid.UUID) (*sn.Group, error) {
	return impl.group.Take(id)
}

func (impl *GroupImpl) GetByName(name string) (*sn.Group, error) {
	return impl.group.Where("name = ?", name).Take()
}

func (impl *GroupImpl) GetUserAllGroup(id uuid.UUID) ([]*sn.UserGroupLink, error) {
	link, err := impl.link.Where("user_group_links.uid = ?", id).Joins("Group").Find()
	if err != nil {
		return nil, err
	}
	return link, nil
}

func (impl *GroupImpl) Count(cond *sn.Condition) (int64, error) {
	return impl.group.Count(cond)
}

func (impl *GroupImpl) Update(column []string, group *sn.Group) error {
	if group == nil {
		return nil
	}
	return impl.group.ID(group.ID).Updates(column, group)
}

func (impl *GroupImpl) Unlink(uid []uuid.UUID, gid []uuid.UUID) error {
	if len(uid) == 0 && len(gid) == 0 {
		return nil
	}
	return impl.link.Transaction(func(tx *gorm.DB) error {
		if uid != nil && gid != nil {
			for _, i := range uid {
				for _, j := range gid {
					if _, err := impl.link.WithTx(tx).Delete("uid = ? AND gid = ?", i, j); err != nil {
						return err
					}
				}
			}
		} else if uid == nil {
			for _, j := range gid {
				if _, err := impl.link.WithTx(tx).Delete("gid = ?", j); err != nil {
					return err
				}
			}
		} else {
			for _, i := range uid {
				if _, err := impl.link.WithTx(tx).Delete("uid = ?", i); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (impl *GroupImpl) Delete(id uuid.UUID) error {
	if _, err := impl.group.DeleteID(id); err != nil {
		return err
	}
	return nil
}

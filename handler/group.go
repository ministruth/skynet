package handler

import (
	"skynet/sn"
	"skynet/sn/impl"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type siteGroup struct {
	tx *gorm.DB
	*impl.ORM[sn.UserGroup]
	link *impl.ORM[sn.UserGroupLink]
}

func NewGroup() sn.SNGroup {
	return &siteGroup{
		tx:   sn.Skynet.GetDB(),
		ORM:  impl.NewORM[sn.UserGroup](nil),
		link: impl.NewORM[sn.UserGroupLink](nil),
	}
}

func (u *siteGroup) WithTx(tx *gorm.DB) sn.SNGroup {
	return &siteGroup{
		tx:   tx,
		ORM:  impl.NewORM[sn.UserGroup](tx),
		link: impl.NewORM[sn.UserGroupLink](tx),
	}
}

func (u *siteGroup) New(name string, note string) (*sn.UserGroup, error) {
	group := &sn.UserGroup{
		Name: name,
		Note: note,
	}
	if err := u.Impl.Create(group); err != nil {
		return nil, err
	}
	return group, nil
}

func (u *siteGroup) Link(uid []uuid.UUID, gid []uuid.UUID) ([]*sn.UserGroupLink, error) {
	if len(uid) == 0 {
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
	if err := u.link.Impl.Creates(group); err != nil {
		return nil, err
	}
	return group, nil
}

func (u *siteGroup) Update(id uuid.UUID, name string, note *string) error {
	group := &sn.UserGroup{
		Name: name,
	}
	var fields []string
	if name != "" {
		fields = append(fields, "name")
	}
	if note != nil {
		fields = append(fields, "note")
		group.Note = *note
	}
	return u.Impl.ID(id).Updates(fields, group)
}

func (u *siteGroup) Delete(id uuid.UUID) (ok bool, err error) {
	err = u.tx.Transaction(func(tx *gorm.DB) error {
		tu := u.WithTx(tx).(*siteGroup)
		_, err = tu.Unlink(uuid.Nil, id)
		if err != nil {
			return err
		}
		ok, err = tu.Delete(id)
		return err
	})
	return
}

func (u *siteGroup) Unlink(uid uuid.UUID, gid uuid.UUID) (int64, error) {
	if uid == uuid.Nil && gid == uuid.Nil {
		return 0, nil
	}
	if gid == uuid.Nil {
		return u.link.Impl.Where("uid = ?", uid).Delete()
	} else if uid == uuid.Nil {
		return u.link.Impl.Where("gid = ?", uid).Delete()
	} else {
		return u.link.Impl.Where("uid = ? AND gid = ?", uid, gid).Delete()
	}
}

func (u *siteGroup) GetGroupAllUser(id uuid.UUID) ([]*sn.User, error) {
	var user []*sn.User
	link, err := u.link.Impl.Where("user_group_links.gid = ?", id).Joins("User").Find()
	if err != nil {
		return nil, err
	}
	for _, v := range link {
		user = append(user, v.User)
	}
	return user, nil
}

func (u *siteGroup) GetByName(name string) (*sn.UserGroup, error) {
	return u.Impl.Where("name = ?", name).Take()
}

func (u *siteGroup) GetUserAllGroup(id uuid.UUID) ([]*sn.UserGroup, error) {
	var group []*sn.UserGroup
	link, err := u.link.Impl.Where("user_group_links.uid = ?", id).Joins("UserGroup").Find()
	if err != nil {
		return nil, err
	}
	for _, v := range link {
		group = append(group, v.UserGroup)
	}
	return group, nil
}

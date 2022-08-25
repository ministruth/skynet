package handler

import (
	"github.com/MXWXZ/skynet/db"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupImpl struct {
	tx    *gorm.DB
	group *db.ORM[db.UserGroup]
	link  *db.ORM[db.UserGroupLink]
}

var Group = &GroupImpl{}

func (u *GroupImpl) WithTx(tx *gorm.DB) *GroupImpl {
	if tx == nil {
		tx = db.DB
	}
	return &GroupImpl{
		tx:    tx,
		group: db.NewORM[db.UserGroup](tx),
		link:  db.NewORM[db.UserGroupLink](tx),
	}
}

// New create new usergroup.
func (u *GroupImpl) New(name string, note string) (*db.UserGroup, error) {
	group := &db.UserGroup{
		Name: name,
		Note: note,
	}
	if err := u.group.Create(group); err != nil {
		return nil, err
	}
	return group, nil
}

// Link link all uid user to all gid group.
//
// Note: For performance reasons, this function will not check whether uid or gid is valid.
func (u *GroupImpl) Link(uid []uuid.UUID, gid []uuid.UUID) ([]*db.UserGroupLink, error) {
	if len(uid) == 0 {
		return nil, nil
	}
	var group []*db.UserGroupLink
	for _, v := range uid {
		for _, g := range gid {
			group = append(group, &db.UserGroupLink{
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

// GetGroupAllUser get group id all users.
func (u *GroupImpl) GetGroupAllUser(id uuid.UUID) ([]*db.User, error) {
	var user []*db.User
	link, err := u.link.Where("user_group_links.gid = ?", id).Joins("User").Find()
	if err != nil {
		return nil, err
	}
	for _, v := range link {
		user = append(user, v.User)
	}
	return user, nil
}

// GetAll get all group by condition.
func (u *GroupImpl) GetAll(cond *db.Condition) ([]*db.UserGroup, error) {
	return u.group.Cond(cond).Find()
}

// Get get group by id.
func (u *GroupImpl) Get(id uuid.UUID) (*db.UserGroup, error) {
	return u.group.Take(id)
}

// GetByName get group by name.
func (u *GroupImpl) GetByName(name string) (*db.UserGroup, error) {
	return u.group.Where("name = ?", name).Take()
}

// GetUserAllGroup get user id all groups.
func (u *GroupImpl) GetUserAllGroup(id uuid.UUID) ([]*db.UserGroup, error) {
	var group []*db.UserGroup
	link, err := u.link.Where("user_group_links.uid = ?", id).Joins("UserGroup").Find()
	if err != nil {
		return nil, err
	}
	for _, v := range link {
		group = append(group, v.UserGroup)
	}
	return group, nil
}

// Count count group by condition.
func (u *GroupImpl) Count(cond *db.Condition) (int64, error) {
	return u.group.Count(cond)
}

// Update update user group infos, properties remain no change if left empty.
func (u *GroupImpl) Update(id uuid.UUID, name string, note *string) error {
	group := &db.UserGroup{
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
	return u.group.ID(id).Updates(fields, group)
}

// Delete delete user group data.
//
// Warning: this function will not delete permission.
func (u *GroupImpl) Delete(id uuid.UUID) (ok bool, err error) {
	err = u.tx.Transaction(func(tx *gorm.DB) error {
		tu := u.WithTx(tx)
		_, err = tu.Unlink(uuid.Nil, id)
		if err != nil {
			return err
		}
		ok, err = tu.group.DeleteID(id)
		return err
	})
	return
}

// DeleteAll delete all user group data.
//
// Warning: this function will not delete permission.
func (u *GroupImpl) DeleteAll() (cnt int64, err error) {
	err = u.tx.Transaction(func(tx *gorm.DB) error {
		tu := u.WithTx(tx)
		cnt, err = tu.group.DeleteAll()
		if err != nil {
			return err
		}
		_, err = tu.link.DeleteAll()
		return err
	})
	return
}

// Unlink delete user data in user group.
// When gid is uuid.Nil, delete user in all group.
// When uid is uuid.Nil, delete group all user.
//
// Warning: this function will not delete permission.
func (u *GroupImpl) Unlink(uid uuid.UUID, gid uuid.UUID) (int64, error) {
	if uid == uuid.Nil && gid == uuid.Nil {
		return 0, nil
	}
	if gid == uuid.Nil {
		return u.link.Where("uid = ?", uid).Delete()
	} else if uid == uuid.Nil {
		return u.link.Where("gid = ?", gid).Delete()
	} else {
		return u.link.Where("uid = ? AND gid = ?", uid, gid).Delete()
	}
}

package handler

import (
	"skynet/sn"
	"skynet/sn/impl"
	"skynet/sn/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type siteUser struct {
	*impl.ORM[sn.User]
}

func NewUser() sn.SNUser {
	return &siteUser{
		ORM: impl.NewORM[sn.User](nil),
	}
}

func (u *siteUser) WithTx(tx *gorm.DB) sn.SNUser {
	return &siteUser{
		ORM: impl.NewORM[sn.User](tx),
	}
}

func (u *siteUser) New(username string, password string,
	avatar *utils.WebpImage) (user *sn.User, newpass string, err error) {
	if password == "" {
		newpass = utils.RandString(8)
	} else {
		newpass = password
	}
	user = &sn.User{
		Username: username,
		Password: impl.HashPass(newpass),
		Avatar:   avatar.Data,
	}
	if err := u.Impl.Create(user); err != nil {
		return nil, "", err
	}
	return
}

func (u *siteUser) Kick(id uuid.UUID) error {
	return impl.DeleteSessions([]uuid.UUID{id})
}

func (u *siteUser) Update(id uuid.UUID, username string, password string,
	avatar *utils.WebpImage, lastTime *time.Time, lastIP string) error {
	user := new(sn.User)
	user.Username = username
	if password != "" {
		user.Password = impl.HashPass(password)
	}
	if avatar != nil {
		user.Avatar = avatar.Data
	}
	if lastTime != nil {
		user.LastLogin = lastTime.UnixMilli()
	}
	user.LastIP = lastIP
	return u.Impl.ID(id).Updates(nil, user)
}

func (u *siteUser) Delete(id uuid.UUID) (ok bool, err error) {
	// kick first
	if err := u.Kick(id); err != nil {
		return false, err
	}

	return u.ORM.Delete(id)
}

func (u *siteUser) GetByName(name string) (*sn.User, error) {
	return u.Impl.Where("username = ?", name).Take()
}

func (u *siteUser) Reset(id uuid.UUID) (string, error) {
	user, err := u.Get(id)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", nil
	}

	// ensure security, kick first
	if err := u.Kick(id); err != nil {
		return "", err
	}

	newpass := utils.RandString(8)
	user.Password = impl.HashPass(newpass)
	if err := u.Impl.Save(user); err != nil {
		return "", err
	}
	return newpass, nil
}

func (u *siteUser) CheckPass(user string, pass string) (*sn.User, int, error) {
	rec, err := u.GetByName(user)
	if err != nil {
		return nil, -1, err
	}
	if rec == nil {
		return nil, 1, nil
	}
	if rec.Password != impl.HashPass(pass) {
		return nil, 2, nil
	}
	return rec, 0, nil
}

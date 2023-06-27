package handler

import (
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

type UserImpl struct {
	orm *sn.ORM[sn.User]
}

func NewUserHandler() sn.UserHandler {
	return &UserImpl{orm: sn.NewORM[sn.User](sn.Skynet.DB)}
}

func (impl *UserImpl) hashPass(pass string) string {
	return utils.MD5(viper.GetString("database.salt_prefix") + pass + viper.GetString("database.salt_suffix"))
}

func (impl *UserImpl) WithTx(tx *gorm.DB) sn.UserHandler {
	return &UserImpl{orm: sn.NewORM[sn.User](tx)}
}

func (impl *UserImpl) New(username string, password string, avatar string) (*sn.User, error) {
	user := &sn.User{
		Username: username,
		Password: impl.hashPass(password),
		Avatar:   avatar,
	}
	if err := impl.orm.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (impl *UserImpl) CheckPass(user string, pass string) (*sn.User, int, error) {
	rec, err := impl.GetByName(user)
	if err != nil {
		return nil, -1, err
	}
	if rec == nil {
		return nil, 1, nil
	}
	if rec.Password != impl.hashPass(pass) {
		return nil, 2, nil
	}
	return rec, 0, nil
}

func (impl *UserImpl) GetAll(cond *sn.Condition) ([]*sn.User, error) {
	return impl.orm.Cond(cond).Find()
}

func (impl *UserImpl) Get(id uuid.UUID) (*sn.User, error) {
	return impl.orm.Take(id)
}

func (impl *UserImpl) GetByName(name string) (*sn.User, error) {
	return impl.orm.Where("username = ?", name).Take()
}

func (impl *UserImpl) Count(cond *sn.Condition) (int64, error) {
	return impl.orm.Count(cond)
}

func (impl *UserImpl) Kick(id uuid.UUID) error {
	return sn.Skynet.Session.Delete([]uuid.UUID{id})
}

func (impl *UserImpl) Reset(id uuid.UUID) (string, error) {
	newPass := utils.RandString(8)
	if err := impl.Update([]string{"password"}, &sn.User{
		GeneralFields: sn.GeneralFields{ID: id},
		Password:      newPass,
	}); err != nil {
		return "", err
	}
	if err := impl.Kick(id); err != nil {
		return "", err
	}
	return newPass, nil
}

func (impl *UserImpl) Update(column []string, user *sn.User) error {
	if user == nil {
		return nil
	}
	kick := false
	if slices.Contains(column, "password") {
		kick = true
		user.Password = impl.hashPass(user.Password)
	}
	if err := impl.orm.ID(user.ID).Updates(column, user); err != nil {
		return err
	}
	if kick {
		return impl.Kick(user.ID)
	}
	return nil
}

func (impl *UserImpl) Delete(id uuid.UUID) error {
	if _, err := impl.orm.DeleteID(id); err != nil {
		return err
	}
	return impl.Kick(id)
}

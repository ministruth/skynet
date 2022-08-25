package handler

import (
	"skynet/db"
	"skynet/utils"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// HashPass returns hashed password.
func HashPass(pass string) string {
	return utils.MD5(viper.GetString("database.salt_prefix") + pass + viper.GetString("database.salt_suffix"))
}

type UserImpl struct {
	orm *db.ORM[db.User]
}

var User = &UserImpl{}

func (u *UserImpl) WithTx(tx *gorm.DB) *UserImpl {
	return &UserImpl{
		orm: db.NewORM[db.User](tx),
	}
}

// New create new user and return created user and created pass, when password is empty, generate random pass,
// by default no user group will be attached.
func (u *UserImpl) New(username string, password string,
	avatar []byte) (user *db.User, newpass string, err error) {
	if password == "" {
		newpass = utils.RandString(8)
	} else {
		newpass = password
	}
	user = &db.User{
		Username: username,
		Password: HashPass(newpass),
		Avatar:   avatar,
	}
	if err := u.orm.Create(user); err != nil {
		return nil, "", err
	}
	return
}

// CheckPass check whether user and pass match.
//
// If error, return nil,-1,err.
//
// If user not found, return nil,1,nil.
//
// If pass not match, return nil,2,nil.
//
// Return user,0,nil if all match.
func (u *UserImpl) CheckPass(user string, pass string) (*db.User, int, error) {
	rec, err := u.GetByName(user)
	if err != nil {
		return nil, -1, err
	}
	if rec == nil {
		return nil, 1, nil
	}
	if rec.Password != HashPass(pass) {
		return nil, 2, nil
	}
	return rec, 0, nil
}

// GetAll get all user by condition.
func (u *UserImpl) GetAll(cond *db.Condition) ([]*db.User, error) {
	return u.orm.Cond(cond).Find()
}

// Get get user by id.
func (u *UserImpl) Get(id uuid.UUID) (*db.User, error) {
	return u.orm.Take(id)
}

// GetByName get user by name.
//
// Return nil,nil when user not found.
func (u *UserImpl) GetByName(name string) (*db.User, error) {
	return u.orm.Where("username = ?", name).Take()
}

// Count count user by condition.
func (u *UserImpl) Count(cond *db.Condition) (int64, error) {
	return u.orm.Count(cond)
}

// Kick kick user id login.
func (u *UserImpl) Kick(id uuid.UUID) error {
	return db.DeleteSessions([]uuid.UUID{id})
}

// Reset reset user password by id, return new password.
//
// Return "",nil when user not found.
func (u *UserImpl) Reset(id uuid.UUID) (string, error) {
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
	user.Password = HashPass(newpass)
	if err := u.orm.Save(user); err != nil {
		return "", err
	}
	return newpass, nil
}

// Update update user infos, properties remain no change if left empty.
func (u *UserImpl) Update(id uuid.UUID, username string, password string,
	avatar []byte, lastTime *time.Time, lastIP string) error {
	user := new(db.User)
	user.Username = username
	if password != "" {
		user.Password = HashPass(password)
	}
	if avatar != nil {
		user.Avatar = avatar
	}
	if lastTime != nil {
		user.LastLogin = lastTime.UnixMilli()
	}
	user.LastIP = lastIP
	return u.orm.ID(id).Updates(nil, user)
}

// Delete delete user data.
//
// Warning: this function will not delete permission or unlink group.
func (u *UserImpl) Delete(id uuid.UUID) (ok bool, err error) {
	// kick first
	if err := u.Kick(id); err != nil {
		return false, err
	}

	ok, err = u.orm.DeleteID(id)
	return
}

// Delete delete all user data.
//
// Warning: this function will not delete permission or unlink group.
func (u *UserImpl) DeleteAll() (int64, error) {
	// kick first
	if err := db.DeleteSessions(nil); err != nil {
		return 0, err
	}

	return u.orm.DeleteAll()
}

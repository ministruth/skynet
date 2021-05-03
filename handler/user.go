package handler

import (
	"context"
	"skynet/sn"
	"skynet/sn/utils"
)

type siteUser struct{}

func NewUser() sn.SNUser {
	return &siteUser{}
}

func (u *siteUser) AddUser(username string, password string, avatar []byte, role sn.UserRole) (string, error) {
	var newpass string
	if password == "" {
		newpass = utils.RandString(8)
	} else {
		newpass = password
	}

	webpAvatar, err := utils.PicFromByte(avatar)
	if err != nil {
		return "", err
	}

	user := sn.Users{
		Username: username,
		Password: HashPass(newpass),
		Avatar:   webpAvatar.Data(),
		Role:     role,
	}
	err = utils.GetDB().Create(&user).Error
	if err != nil {
		return "", err
	}
	return newpass, nil
}

func (u *siteUser) EditUser(id int, username string, password string, role sn.UserRole, avatar []byte, kick bool) error {
	var err error
	if kick {
		err = utils.DeleteSessionsByID(id)
		if err != nil {
			return err
		}
	}
	if username == "" && password == "" && role == sn.RoleEmpty {
		return nil
	}

	var webpAvatar *utils.WebpImage
	if avatar != nil {
		webpAvatar, err = utils.PicFromByte(avatar)
		if err != nil {
			return err
		}
	}

	var rec sn.Users
	err = utils.GetDB().First(&rec, id).Error
	if err != nil {
		return err
	}
	if username != "" {
		rec.Username = username
	}
	if password != "" {
		rec.Password = HashPass(password)
	}
	if role != sn.RoleEmpty {
		rec.Role = role
	}
	if avatar != nil {
		rec.Avatar = webpAvatar.Data()
	}
	return utils.GetDB().Save(&rec).Error
}

func (u *siteUser) ResetUser(username string) (string, error) {
	var rec sn.Users
	err := utils.GetDB().Where("username = ?", username).First(&rec).Error
	if err != nil {
		return "", err
	}

	// ensure security, kick first
	err = utils.DeleteSessionsByID(int(rec.ID))
	if err != nil {
		return "", err
	}

	newpass := utils.RandString(8)
	rec.Password = HashPass(newpass)
	err = utils.GetDB().Save(&rec).Error
	if err != nil {
		return "", err
	}

	return newpass, nil
}

func (u *siteUser) ResetAllUser() (map[string]string, error) {
	var rec []sn.Users
	ret := make(map[string]string)
	err := utils.GetDB().Find(&rec).Error
	if err != nil {
		return nil, err
	}
	if len(rec) == 0 {
		return ret, nil
	}

	// ensure security, kick first
	err = utils.GetRedis().FlushDB(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	for i := range rec {
		newpass := utils.RandString(8)
		rec[i].Password = HashPass(newpass)
		ret[rec[i].Username] = newpass
	}

	err = utils.GetDB().Save(&rec).Error
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (u *siteUser) GetUser() ([]sn.Users, error) {
	var users []sn.Users
	err := utils.GetDB().Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

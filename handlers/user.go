package handlers

import (
	"context"
	"skynet/db"
	"skynet/utils"
)

func AddUser(username string, password string, avatar []byte) (string, error) {
	var newpass string
	if password == "" {
		newpass = utils.RandString(8)
	} else {
		newpass = password
	}

	webpAvatar, err := utils.ConvertPicture(avatar)
	if err != nil {
		return "", err
	}

	user := db.Users{
		Username: username,
		Password: HashPass(newpass),
		Avatar:   webpAvatar,
	}
	err = db.GetDB().Create(&user).Error
	if err != nil {
		return "", err
	}
	return newpass, nil
}

func ResetUser(username string) (string, error) {
	var rec db.Users
	err := db.GetDB().Where("username = ?", username).First(&rec).Error
	if err != nil {
		return "", err
	}

	// ensure security, delete first
	err = utils.DeleteSessionsByID(int(rec.ID))
	if err != nil {
		return "", err
	}

	newpass := utils.RandString(8)
	rec.Password = HashPass(newpass)
	err = db.GetDB().Save(&rec).Error
	if err != nil {
		return "", err
	}

	return newpass, nil
}

func ResetAllUser() (map[string]string, error) {
	var rec []db.Users
	ret := make(map[string]string)
	err := db.GetDB().Find(&rec).Error
	if err != nil {
		return nil, err
	}
	if len(rec) == 0 {
		return ret, nil
	}

	// ensure security, delete first
	err = db.GetRedis().FlushDB(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	for i := range rec {
		newpass := utils.RandString(8)
		rec[i].Password = HashPass(newpass)
		ret[rec[i].Username] = newpass
	}

	err = db.GetDB().Save(&rec).Error
	if err != nil {
		return nil, err
	}
	return ret, nil
}

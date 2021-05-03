package handler

import (
	"errors"

	"skynet/sn"
	"skynet/sn/utils"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func HashPass(pass string) string {
	return utils.MD5(viper.GetString("database.salt_prefix") + pass + viper.GetString("database.salt_suffix"))
}

func CheckUserPass(user string, pass string) (*sn.Users, int, error) {
	var rec sn.Users
	err := utils.GetDB().Where("username = ?", user).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 1, nil
	} else if err != nil {
		return nil, -1, err
	}
	if rec.Password != HashPass(pass) {
		return nil, 2, nil
	}
	return &rec, 0, nil
}

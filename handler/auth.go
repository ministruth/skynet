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

func CheckUserPass(user string, pass string) (*sn.User, int, error) {
	rec, err := sn.Skynet.User.GetByUsername(user)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 1, nil
	} else if err != nil {
		return nil, -1, err
	}
	if rec.Password != HashPass(pass) {
		return nil, 2, nil
	}
	return rec, 0, nil
}

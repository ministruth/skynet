package handler

import (
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils/log"
)

func Init() {
	sn.Skynet.User = NewUserHandler()
	sn.Skynet.Group = NewGroupHandler()
	sn.Skynet.Permission = NewPermissionHandler()
	sn.Skynet.Notification = NewNotificationHandler()
	sn.Skynet.Setting = NewSettingHandler()
	if err := sn.Skynet.Setting.BuildCache(); err != nil {
		log.NewEntry(err).Fatal("Failed to build setting cache")
	}
}

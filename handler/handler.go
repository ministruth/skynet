package handler

import "github.com/MXWXZ/skynet/utils/log"

func Init() {
	User = User.WithTx(nil)
	Group = Group.WithTx(nil)
	Notification = Notification.WithTx(nil)
	Setting = Setting.WithTx(nil)
	if err := Setting.BuildCache(); err != nil {
		log.NewEntry(err).Fatal("Failed to build setting cache")
	}
	Permission = Permission.WithTx(nil)
}

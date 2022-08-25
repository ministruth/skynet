package handler

import "skynet/utils/log"

func Init() {
	User = User.WithTx(nil)
	Group = Group.WithTx(nil)
	Notification = Notification.WithTx(nil)
	Setting = Setting.WithTx(nil)
	if err := Setting.BuildCache(); err != nil {
		log.NewEntry(err).Fatal("Failed to build setting cache")
	}
	Permission = Permission.WithTx(nil)

	s := pluginHelper{}
	_, err := s.Eval(`
	import "skynet/handler"

	func main() {
		handler.Setting.Get("plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa")
	}
	`)
	if err != nil {
		panic(err)
	}
}

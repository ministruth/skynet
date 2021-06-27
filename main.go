package main

import (
	"skynet/cmd"
	"skynet/handler"
	"skynet/sn"
)

func init() {
	sn.Skynet.User = handler.NewUser()
	sn.Skynet.Notification = handler.NewNotification()
}

func main() {
	cmd.Execute()
}

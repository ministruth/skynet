package main

import (
	"skynet/cmd"
	"skynet/handler"
	"skynet/sn"
)

func init() {
	sn.Skynet.User = handler.NewUser()
}

func main() {
	cmd.Execute()
}

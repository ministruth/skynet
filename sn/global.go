package sn

type SNGlobal struct {
	API       SNAPI
	Page      SNPage
	Plugin    SNPlugin
	Setting   SNSetting
	User      SNUser
	DB        SNDB
	Redis     SNDB
	Session   SNDB
	ShareData map[string]interface{}
}

const VERSION = "1.0.0"

var Skynet SNGlobal

package sn

func init() {
	Skynet.SharedData = make(map[string]interface{})
}

type SNGlobal struct {
	API        SNAPI
	Page       SNPage
	Plugin     SNPlugin
	Setting    SNSetting
	User       SNUser
	DB         SNDB
	Redis      SNDB
	Session    SNDB
	SharedData map[string]interface{}
}

const VERSION = "1.0.0"

var Skynet SNGlobal

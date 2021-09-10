package api

import "skynet/sn"

var api = []*sn.SNAPIItem{
	{
		Path:   "/signin",
		Method: sn.APIPost,
		Role:   sn.RoleEmpty,
		Func:   APISignIn,
	},
	{
		Path:   "/signout",
		Method: sn.APIGet,
		Role:   sn.RoleUser,
		Func:   APISignOut,
	},
	{
		Path:   "/user",
		Method: sn.APIGet,
		Role:   sn.RoleAdmin,
		Func:   APIGetUser,
	},
	{
		Path:   "/user/:id",
		Method: sn.APIPatch,
		Role:   sn.RoleUser,
		Func:   APIUpdateUser,
	},
	{
		Path:   "/user",
		Method: sn.APIPost,
		Role:   sn.RoleAdmin,
		Func:   APIAddUser,
	},
	{
		Path:   "/user/:id",
		Method: sn.APIDelete,
		Role:   sn.RoleAdmin,
		Func:   APIDeleteUser,
	},
	{
		Path:   "/reload",
		Method: sn.APIGet,
		Role:   sn.RoleAdmin,
		Func:   APIReload,
	},
	{
		Path:   "/plugin",
		Method: sn.APIGet,
		Role:   sn.RoleAdmin,
		Func:   APIGetPlugin,
	},
	{
		Path:   "/plugin",
		Method: sn.APIPost,
		Role:   sn.RoleAdmin,
		Func:   APIAddPlugin,
	},
	{
		Path:   "/plugin/:id",
		Method: sn.APIPatch,
		Role:   sn.RoleAdmin,
		Func:   APIEnablePlugin,
	},
	{
		Path:   "/notification",
		Method: sn.APIGet,
		Role:   sn.RoleUser,
		Func:   APIGetNotification,
	},
	{
		Path:   "/notification",
		Method: sn.APIDelete,
		Role:   sn.RoleAdmin,
		Func:   APIDeleteNotification,
	},
	{
		Path:   "/setting/navbar",
		Method: sn.APIPost,
		Role:   sn.RoleAdmin,
		Func:   APIUpdateNavbar,
	},
}

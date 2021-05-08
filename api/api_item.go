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
		Method: sn.APIPatch,
		Role:   sn.RoleUser,
		Func:   APIEditUser,
	},
	{
		Path:   "/user",
		Method: sn.APIPost,
		Role:   sn.RoleAdmin,
		Func:   APIAddUser,
	},
	{
		Path:   "/user",
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
		Method: sn.APIPatch,
		Role:   sn.RoleAdmin,
		Func:   APIEditPlugin,
	},
}

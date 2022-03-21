package api

import (
	"skynet/sn"

	"github.com/google/uuid"
)

func initAPI() []*sn.SNAPIItem {
	return []*sn.SNAPIItem{
		{
			Path:   "/ping",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermGuestID),
				Perm: sn.PermAll,
			},
			Func: APIPing,
		},
		{
			Path:   "/signin",
			Method: sn.APIPost,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermGuestID),
				Perm: sn.PermAll,
			},
			Func: APISignIn,
		},
		{
			Path:   "/signout",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermUserID),
				Perm: sn.PermAll,
			},
			Func: APISignOut,
		},
		{
			Path:   "/reload",
			Method: sn.APIPost,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageSystemID),
				Perm: sn.PermExecute,
			},
			Func: APIReload,
		},
		{
			Path:   "/access",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermGuestID),
				Perm: sn.PermAll,
			},
			Func: APIGetAccess,
		},
		{
			Path:   "/token",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermGuestID),
				Perm: sn.PermAll,
			},
			Func: APIGetCSRFToken,
		},
		{
			Path:   "/setting/public",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermGuestID),
				Perm: sn.PermAll,
			},
			Func: APIGetPublicSetting,
		},
		{
			Path:   "/menu",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermUserID),
				Perm: sn.PermAll,
			},
			Func: APIGetMenu,
		},
		{
			Path:   "/notification",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageNotificationID),
				Perm: sn.PermRead,
			},
			Func: APIGetNotification,
		},
		{
			Path:   "/notification",
			Method: sn.APIDelete,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageNotificationID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteNotification,
		},
		{
			Path:   "/user",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetUser,
		},
		{
			Path:   "/user",
			Method: sn.APIPost,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIAddUser,
		},
		{
			Path:   "/user/:id",
			Method: sn.APIDelete,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteUser,
		},
		{
			Path:   "/group",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageGroupID),
				Perm: sn.PermRead,
			},
			Func: APIGetGroup,
		},
		{
			Path:   "/group",
			Method: sn.APIPost,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageGroupID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIAddGroup,
		},
		{
			Path:   "/group/:id",
			Method: sn.APIDelete,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageGroupID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteGroup,
		},
		{
			Path:   "/group/:id",
			Method: sn.APIPut,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageGroupID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIPutGroup,
		},
		{
			Path:   "/plugin",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManagePluginID),
				Perm: sn.PermRead,
			},
			Func: APIGetPlugin,
		},
		{
			Path:   "/plugin",
			Method: sn.APIPost,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManagePluginID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIAddPlugin,
		},
		{
			Path:   "/plugin/entry",
			Method: sn.APIGet,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermGuestID),
				Perm: sn.PermAll,
			},
			Func: APIGetPluginEntry,
		},
		{
			Path:   "/plugin/:id",
			Method: sn.APIPut,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManagePluginID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIPutPlugin,
		},
		{
			Path:   "/plugin/:id",
			Method: sn.APIDelete,
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManagePluginID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeletePlugin,
		},
	}
}

func initMenu() []*sn.SNMenu {
	return []*sn.SNMenu{
		{
			ID:   uuid.MustParse("7bd9a6d3-db3d-4954-89ca-d5b1f3d9974f"),
			Name: "menu.dashboard",
			Path: "/dashboard",
			Icon: "DashboardOutlined",
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermUserID),
				Perm: sn.PermAll,
			},
		},
		{
			ID:   uuid.MustParse("d00d36d0-6068-4447-ab04-f82ce893c04e"),
			Name: "menu.service",
			Icon: "FunctionOutlined",
		},
		{
			ID:   uuid.MustParse("cca5b3b0-40a3-465c-8b08-91f3e8d3b14d"),
			Name: "menu.plugin",
			Icon: "ApiOutlined",
			Children: []*sn.SNMenu{
				{
					ID:   uuid.MustParse("251a16e1-655b-4716-8766-cd2bc66d6309"),
					Name: "menu.plugin.manage",
					Path: "/plugin",
					Perm: &sn.SNPerm{
						ID:   sn.Skynet.GetID(sn.PermManagePluginID),
						Perm: sn.PermRead,
					},
				},
			},
		},
		{
			ID:   uuid.MustParse("4d6c60d7-9c2a-44f0-b85a-346425df792f"),
			Name: "menu.user",
			Icon: "UserOutlined",
			Children: []*sn.SNMenu{
				{
					ID:   uuid.MustParse("0d2165b9-e08b-429f-ad4e-420472083e0f"),
					Name: "menu.user.user",
					Path: "/user",
					Perm: &sn.SNPerm{
						ID:   sn.Skynet.GetID(sn.PermManageUserID),
						Perm: sn.PermRead,
					},
				},
				{
					ID:   uuid.MustParse("03e3caeb-9008-4e5c-9e19-c11d6b567aa7"),
					Name: "menu.user.group",
					Path: "/group",
					Perm: &sn.SNPerm{
						ID:   sn.Skynet.GetID(sn.PermManageGroupID),
						Perm: sn.PermRead,
					},
				},
			},
		},
		{
			ID:   uuid.MustParse("06c21cbc-b43f-4b43-a633-8baf2221493f"),
			Name: "menu.notification",
			Path: "/notification",
			Icon: "NotificationOutlined",
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageNotificationID),
				Perm: sn.PermRead,
			},
		},
		{
			ID:   uuid.MustParse("4b9df963-c540-48f4-9bfb-500f06ecfef0"),
			Name: "menu.system",
			Path: "/system",
			Icon: "SettingOutlined",
			Perm: &sn.SNPerm{
				ID:   sn.Skynet.GetID(sn.PermManageSystemID),
				Perm: sn.PermRead,
			},
		},
	}
}

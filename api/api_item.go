package api

import (
	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/handler"
	"github.com/google/uuid"
)

func initAPI() []*APIItem {
	PID := db.GetDefaultID
	return []*APIItem{
		{
			Path:   "/ping",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermGuestID),
				Perm: db.PermAll,
			},
			Func: APIPing,
		},
		{
			Path:   "/signin",
			Method: APIPost,
			Perm: &handler.Perm{
				ID:   PID(db.PermGuestID),
				Perm: db.PermAll,
			},
			Func: APISignIn,
		},
		{
			Path:   "/signout",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermUserID),
				Perm: db.PermAll,
			},
			Func: APISignOut,
		},
		{
			Path:   "/token",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermGuestID),
				Perm: db.PermAll,
			},
			Func: APIGetCSRFToken,
		},
		{
			Path:   "/reload",
			Method: APIPost,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageSystemID),
				Perm: db.PermExecute,
			},
			Func: APIReload,
		},
		{
			Path:   "/access",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermGuestID),
				Perm: db.PermAll,
			},
			Func: APIGetAccess,
		},
		{
			Path:   "/menu",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermUserID),
				Perm: db.PermAll,
			},
			Func: APIGetMenu,
		},
		{
			Path:   "/setting/public",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermGuestID),
				Perm: db.PermAll,
			},
			Func: APIGetPublicSetting,
		},
		{
			Path:   "/notification",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageNotificationID),
				Perm: db.PermRead,
			},
			Func: APIGetNotification,
		},
		{
			Path:   "/notification",
			Method: APIDelete,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageNotificationID),
				Perm: db.PermWriteExecute,
			},
			Func: APIDeleteNotification,
		},
		{
			Path:   "/user",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageUserID),
				Perm: db.PermRead,
			},
			Func: APIGetUser,
		},
		{
			Path:   "/user",
			Method: APIPost,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageUserID),
				Perm: db.PermWriteExecute,
			},
			Func: APIAddUser,
		},
		{
			Path:   "/user/:id",
			Method: APIDelete,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageUserID),
				Perm: db.PermWriteExecute,
			},
			Func: APIDeleteUser,
		},
		{
			Path:   "/group",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageGroupID),
				Perm: db.PermRead,
			},
			Func: APIGetGroup,
		},
		{
			Path:   "/group",
			Method: APIPost,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageGroupID),
				Perm: db.PermWriteExecute,
			},
			Func: APIAddGroup,
		},
		{
			Path:   "/group/:id",
			Method: APIDelete,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageGroupID),
				Perm: db.PermWriteExecute,
			},
			Func: APIDeleteGroup,
		},
		{
			Path:   "/group/:id",
			Method: APIPut,
			Perm: &handler.Perm{
				ID:   PID(db.PermManageGroupID),
				Perm: db.PermWriteExecute,
			},
			Func: APIPutGroup,
		},
		{
			Path:   "/plugin",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermManagePluginID),
				Perm: db.PermRead,
			},
			Func: APIGetPlugin,
		},
		// {
		// 	Path:   "/plugin",
		// 	Method: sn.APIPost,
		// 	Perm: &handler.Perm{
		// 		ID:   PID(sn.PermManagePluginID),
		// 		Perm: sn.PermWriteExecute,
		// 	},
		// 	Func: APIAddPlugin,
		// },
		{
			Path:   "/plugin/entry",
			Method: APIGet,
			Perm: &handler.Perm{
				ID:   PID(db.PermGuestID),
				Perm: db.PermAll,
			},
			Func: APIGetPluginEntry,
		},
		{
			Path:   "/plugin/:id",
			Method: APIPut,
			Perm: &handler.Perm{
				ID:   PID(db.PermManagePluginID),
				Perm: db.PermWriteExecute,
			},
			Func: APIPutPlugin,
		},
		// {
		// 	Path:   "/plugin/:id",
		// 	Method: sn.APIDelete,
		// 	Perm: &handler.Perm{
		// 		ID:   PID(sn.PermManagePluginID),
		// 		Perm: sn.PermWriteExecute,
		// 	},
		// 	Func: APIDeletePlugin,
		// },
	}
}

func initMenu() []*MenuItem {
	UID := uuid.MustParse
	PID := db.GetDefaultID
	return []*MenuItem{
		{
			ID:   UID("7bd9a6d3-db3d-4954-89ca-d5b1f3d9974f"),
			Name: "menu.dashboard",
			Path: "/dashboard",
			Icon: "DashboardOutlined",
			Perm: &handler.Perm{
				ID:   PID(db.PermUserID),
				Perm: db.PermAll,
			},
		},
		{
			ID:   UID("d00d36d0-6068-4447-ab04-f82ce893c04e"),
			Name: "menu.service",
			Icon: "FunctionOutlined",
		},
		{
			ID:   UID("cca5b3b0-40a3-465c-8b08-91f3e8d3b14d"),
			Name: "menu.plugin",
			Icon: "ApiOutlined",
			Children: []*MenuItem{
				{
					ID:   UID("251a16e1-655b-4716-8766-cd2bc66d6309"),
					Name: "menu.plugin.manage",
					Path: "/plugin",
					Perm: &handler.Perm{
						ID:   PID(db.PermManagePluginID),
						Perm: db.PermRead,
					},
				},
			},
		},
		{
			ID:        UID("4d6c60d7-9c2a-44f0-b85a-346425df792f"),
			Name:      "menu.user",
			OmitEmpty: true,
			Icon:      "UserOutlined",
			Children: []*MenuItem{
				{
					ID:   UID("0d2165b9-e08b-429f-ad4e-420472083e0f"),
					Name: "menu.user.user",
					Path: "/user",
					Perm: &handler.Perm{
						ID:   PID(db.PermManageUserID),
						Perm: db.PermRead,
					},
				},
				{
					ID:   UID("03e3caeb-9008-4e5c-9e19-c11d6b567aa7"),
					Name: "menu.user.group",
					Path: "/group",
					Perm: &handler.Perm{
						ID:   PID(db.PermManageGroupID),
						Perm: db.PermRead,
					},
				},
			},
		},
		{
			ID:   UID("06c21cbc-b43f-4b43-a633-8baf2221493f"),
			Name: "menu.notification",
			Path: "/notification",
			Icon: "NotificationOutlined",
			Perm: &handler.Perm{
				ID:   PID(db.PermManageNotificationID),
				Perm: db.PermRead,
			},
		},
		{
			ID:   UID("4b9df963-c540-48f4-9bfb-500f06ecfef0"),
			Name: "menu.system",
			Path: "/system",
			Icon: "SettingOutlined",
			Perm: &handler.Perm{
				ID:   PID(db.PermManageSystemID),
				Perm: db.PermRead,
			},
		},
	}
}

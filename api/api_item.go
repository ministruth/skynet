package api

import (
	"github.com/MXWXZ/skynet/sn"
	"github.com/google/uuid"
)

func initAPI() []*sn.APIItem {
	PID := sn.Skynet.ID.Get
	return []*sn.APIItem{
		{
			Path:   "/ping",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID: PID(sn.PermGuestID),
			},
			Func: APIPing,
		},
		{
			Path:   "/signin",
			Method: sn.APIPost,
			Perm: &sn.PermEntry{
				ID: PID(sn.PermGuestID),
			},
			Func: APISignIn,
		},
		{
			Path:   "/signout",
			Method: sn.APIPost,
			Perm: &sn.PermEntry{
				ID: PID(sn.PermUserID),
			},
			Func: APISignOut,
		},
		{
			Path:   "/token",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID: PID(sn.PermGuestID),
			},
			Func: APIGetCSRFToken,
		},
		{
			Path:   "/shutdown",
			Method: sn.APIPost,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageSystemID),
				Perm: sn.PermExecute,
			},
			Func: APIShutdown,
		},
		{
			Path:   "/access",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID: PID(sn.PermGuestID),
			},
			Func: APIGetAccess,
		},
		{
			Path:   "/menu",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID: PID(sn.PermUserID),
			},
			Func: APIGetMenu,
		},
		{
			Path:   "/setting/public",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID: PID(sn.PermGuestID),
			},
			Func: APIGetPublicSetting,
		},
		{
			Path:   "/notification",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageNotificationID),
				Perm: sn.PermRead,
			},
			Func: APIGetNotification,
		},
		{
			Path:   "/notification/unread",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageNotificationID),
				Perm: sn.PermRead,
			},
			Func: APIGetUnreadNotification,
		},
		{
			Path:   "/notification",
			Method: sn.APIDelete,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageNotificationID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteNotification,
		},
		{
			Path:   "/user",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetUsers,
		},
		{
			Path:   "/user",
			Method: sn.APIPost,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIAddUser,
		},
		{
			Path:   "/user",
			Method: sn.APIDelete,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteUsers,
		},
		{
			Path:   "/user/:id",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetUser,
		},
		{
			Path:   "/user/:id",
			Method: sn.APIPut,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIPutUser,
		},
		{
			Path:   "/user/:id",
			Method: sn.APIDelete,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteUser,
		},
		{
			Path:   "/user/:id/group",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetUserGroup,
		},
		{
			Path:   "/user/:id/kick",
			Method: sn.APIPost,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermExecute,
			},
			Func: APIKickUser,
		},
		{
			Path:   "/user/:id/permission",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetUserPermission,
		},
		{
			Path:   "/user/:id/permission",
			Method: sn.APIPut,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIPutUserPermission,
		},
		{
			Path:   "/group",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetGroups,
		},
		{
			Path:   "/group",
			Method: sn.APIPost,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIAddGroup,
		},
		{
			Path:   "/group",
			Method: sn.APIDelete,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteGroups,
		},
		{
			Path:   "/group/:id",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetGroup,
		},
		{
			Path:   "/group/:id",
			Method: sn.APIDelete,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteGroup,
		},
		{
			Path:   "/group/:id",
			Method: sn.APIPut,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIPutGroup,
		},
		{
			Path:   "/group/:id/user",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetGroupUser,
		},
		{
			Path:   "/group/:id/user/:uid",
			Method: sn.APIDelete,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteGroupUser,
		},
		{
			Path:   "/group/:id/user",
			Method: sn.APIPost,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIAddGroupUsers,
		},
		{
			Path:   "/group/:id/user",
			Method: sn.APIDelete,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeleteGroupUsers,
		},
		{
			Path:   "/group/:id/permission",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetGroupPermission,
		},
		{
			Path:   "/group/:id/permission",
			Method: sn.APIPut,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageUserID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIPutGroupPermission,
		},
		{
			Path:   "/permission",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermUserID),
				Perm: sn.PermRead,
			},
			Func: APIGetPermission,
		},
		{
			Path:   "/plugin",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManagePluginID),
				Perm: sn.PermRead,
			},
			Func: APIGetPlugin,
		},
		{
			Path:   "/plugin/entry",
			Method: sn.APIGet,
			Perm: &sn.PermEntry{
				ID: PID(sn.PermGuestID),
			},
			Func: APIGetPluginEntry,
		},
		{
			Path:   "/plugin/:id",
			Method: sn.APIPut,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManagePluginID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIPutPlugin,
		},
		{
			Path:   "/plugin/:id",
			Method: sn.APIDelete,
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManagePluginID),
				Perm: sn.PermWriteExecute,
			},
			Func: APIDeletePlugin,
		},
	}
}

func initMenu() []*sn.MenuItem {
	UID := uuid.MustParse
	PID := sn.Skynet.ID.Get
	return []*sn.MenuItem{
		{
			ID:   UID("7bd9a6d3-db3d-4954-89ca-d5b1f3d9974f"),
			Name: "menu.dashboard",
			Path: "/dashboard",
			Icon: "DashboardOutlined",
			Perm: &sn.PermEntry{
				ID: PID(sn.PermUserID),
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
			Children: []*sn.MenuItem{
				{
					ID:   UID("251a16e1-655b-4716-8766-cd2bc66d6309"),
					Name: "menu.plugin.manage",
					Path: "/plugin",
					Perm: &sn.PermEntry{
						ID:   PID(sn.PermManagePluginID),
						Perm: sn.PermRead,
					},
				},
			},
		},
		{
			ID:        UID("4d6c60d7-9c2a-44f0-b85a-346425df792f"),
			Name:      "menu.user",
			OmitEmpty: true,
			Icon:      "UserOutlined",
			Children: []*sn.MenuItem{
				{
					ID:   UID("0d2165b9-e08b-429f-ad4e-420472083e0f"),
					Name: "menu.user.user",
					Path: "/user",
					Perm: &sn.PermEntry{
						ID:   PID(sn.PermManageUserID),
						Perm: sn.PermRead,
					},
				},
				{
					ID:   UID("03e3caeb-9008-4e5c-9e19-c11d6b567aa7"),
					Name: "menu.user.group",
					Path: "/group",
					Perm: &sn.PermEntry{
						ID:   PID(sn.PermManageUserID),
						Perm: sn.PermRead,
					},
				},
			},
		},
		{
			ID:   UID("06c21cbc-b43f-4b43-a633-8baf2221493f"),
			Name: "menu.notification",
			Path: "/notification",
			Icon: "NotificationOutlined",
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageNotificationID),
				Perm: sn.PermRead,
			},
			BadgeFunc: sn.Skynet.Notification.GetUnread,
		},
		{
			ID:   UID("4b9df963-c540-48f4-9bfb-500f06ecfef0"),
			Name: "menu.system",
			Path: "/system",
			Icon: "SettingOutlined",
			Perm: &sn.PermEntry{
				ID:   PID(sn.PermManageSystemID),
				Perm: sn.PermRead,
			},
		},
	}
}

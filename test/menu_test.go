package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMenu(t *testing.T) {
	tests := []testCase{
		{
			name:     "Get Menu Fail",
			url:      "/menu",
			httpCode: 403,
		},
		{
			name:   "Get Menu User",
			url:    "/menu",
			client: loginNormal(t),
			data: []msa{
				{"name": "Dashboard", "path": "/dashboard", "icon": "DashboardOutlined", "children": []msa{}},
				{"name": "Service", "path": "", "icon": "FunctionOutlined", "children": []msa{}},
				{"name": "Plugin", "path": "", "icon": "ApiOutlined", "children": []msa{}},
			},
		},
		{
			name:   "Get Menu Root",
			url:    "/menu",
			client: loginRoot(t),
			data: []msa{
				{"name": "Dashboard", "path": "/dashboard", "icon": "DashboardOutlined", "children": []msa{}},
				{"name": "Service", "path": "", "icon": "FunctionOutlined", "children": []msa{}},
				{"name": "Plugin", "path": "", "icon": "ApiOutlined", "children": []msa{
					{"name": "Manage", "path": "/plugin", "icon": "", "children": []msa{}},
				}},
				{"name": "User", "path": "", "icon": "UserOutlined", "children": []msa{
					{"name": "User", "path": "/user", "icon": "", "children": []msa{}},
					{"name": "Group", "path": "/group", "icon": "", "children": []msa{}},
				}},
				{"name": "Notification", "path": "/notification", "icon": "NotificationOutlined", "children": []msa{}},
				{"name": "System", "path": "/system", "icon": "SettingOutlined", "children": []msa{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test(t)
			assert.Nil(t, err)
		})
	}
}

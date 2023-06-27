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
				{"name": "Dashboard", "badge": 0, "path": "/dashboard", "icon": "DashboardOutlined", "children": []msa{}},
				{"name": "Service", "badge": 0, "path": "", "icon": "FunctionOutlined", "children": []msa{}},
				{"name": "Plugin", "badge": 0, "path": "", "icon": "ApiOutlined", "children": []msa{}},
			},
		},
		{
			name:   "Get Menu Root",
			url:    "/menu",
			client: loginRoot(t),
			data: []msa{
				{"name": "Dashboard", "badge": 0, "path": "/dashboard", "icon": "DashboardOutlined", "children": []msa{}},
				{"name": "Service", "badge": 0, "path": "", "icon": "FunctionOutlined", "children": []msa{}},
				{"name": "Plugin", "badge": 0, "path": "", "icon": "ApiOutlined", "children": []msa{
					{"name": "Manage", "badge": 0, "path": "/plugin", "icon": "", "children": []msa{}},
				}},
				{"name": "User", "badge": 0, "path": "", "icon": "UserOutlined", "children": []msa{
					{"name": "User", "badge": 0, "path": "/user", "icon": "", "children": []msa{}},
					{"name": "Group", "badge": 0, "path": "/group", "icon": "", "children": []msa{}},
				}},
				{"name": "Notification", "badge": 8, "path": "/notification", "icon": "NotificationOutlined", "children": []msa{}},
				{"name": "System", "badge": 0, "path": "/system", "icon": "SettingOutlined", "children": []msa{}},
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

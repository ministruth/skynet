package handler

import (
	"os"
	"testing"

	"github.com/MXWXZ/skynet/db"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
)

var (
	dataPL = []*db.PermissionList{
		{
			GeneralFields: db.GeneralFields{
				ID: uuid.New(),
			},
			Name: "perm1",
			Note: "p_abc",
		},
		{
			GeneralFields: db.GeneralFields{
				ID: uuid.New(),
			},
			Name: "perm2",
			Note: "p_def",
		},
		{
			GeneralFields: db.GeneralFields{
				ID: uuid.New(),
			},
			Name: "perm3",
			Note: "p_ghi",
		},
	}
	dataU = []*db.User{
		{
			GeneralFields: db.GeneralFields{
				ID: uuid.New(),
			},
			Username: "user1",
		},
		{
			GeneralFields: db.GeneralFields{
				ID: uuid.New(),
			},
			Username: "user2",
		},
		{
			GeneralFields: db.GeneralFields{
				ID: uuid.New(),
			},
			Username: "user3",
		},
	}
	dataG = []*db.UserGroup{
		{
			GeneralFields: db.GeneralFields{
				ID: uuid.New(),
			},
			Name: "group1",
			Note: "g_abc",
		},
		{
			GeneralFields: db.GeneralFields{
				ID: uuid.New(),
			},
			Name: "group2",
			Note: "g_def",
		},
		{
			GeneralFields: db.GeneralFields{
				ID: uuid.New(),
			},
			Name: "group3",
			Note: "g_ghi",
		},
	}
	dataUGL = []*db.UserGroupLink{
		{
			UID: dataU[0].ID,
			GID: dataG[0].ID,
		},
		{
			UID: dataU[1].ID,
			GID: dataG[0].ID,
		},
	}

	dataP = []*db.Permission{
		{
			GID:            dataG[0].ID,
			PID:            dataPL[0].ID,
			Perm:           db.PermAll,
			PermissionList: dataPL[0],
		},
		{
			GID:            dataG[0].ID,
			PID:            dataPL[1].ID,
			Perm:           db.PermRead,
			PermissionList: dataPL[1],
		},
		{
			GID:            dataG[1].ID,
			PID:            dataPL[1].ID,
			Perm:           db.PermWrite,
			PermissionList: dataPL[1],
		},
		{
			GID:            dataG[2].ID,
			PID:            dataPL[2].ID,
			Perm:           db.PermExecute,
			PermissionList: dataPL[2],
		},
		{
			UID:            dataU[0].ID,
			PID:            dataPL[0].ID,
			Perm:           db.PermNone,
			PermissionList: dataPL[0],
		},
		{
			UID:            dataU[1].ID,
			PID:            dataPL[0].ID,
			Perm:           db.PermRead,
			PermissionList: dataPL[0],
		},
		{
			UID:            dataU[1].ID,
			PID:            dataPL[2].ID,
			Perm:           db.PermWrite,
			PermissionList: dataPL[2],
		},
		{
			UID:            dataU[2].ID,
			PID:            dataPL[1].ID,
			Perm:           db.PermAll,
			PermissionList: dataPL[1],
		},
	}
)

func setup() {
	viper.SetDefault("database.path", "file::memory:")
	viper.SetDefault("database.type", "sqlite")
	db.NewDB()
	Permission = Permission.WithTx(nil)

	if err := db.DB.Create(&dataPL).Error; err != nil {
		panic(err)
	}
	if err := db.DB.Omit("PermissionList").Create(&dataP).Error; err != nil {
		panic(err)
	}
	for _, v := range dataP {
		v.CreatedAt = 0
		v.UpdatedAt = 0
		v.PermissionList.CreatedAt = 0
		v.PermissionList.UpdatedAt = 0
	}
	db.DB.Logger = logger.Default.LogMode(logger.Info)
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func Test_Permission_GetAll(t *testing.T) {
	type args struct {
		uid  uuid.UUID
		gid  uuid.UUID
		join bool
	}
	tests := []struct {
		name     string
		args     args
		wantPerm []*db.Permission
		wantErr  bool
	}{
		{
			name: "GetAll_group0",
			args: args{
				gid:  dataG[0].ID,
				join: true,
			},
			wantPerm: []*db.Permission{
				dataP[0],
				dataP[1],
			},
		},
		{
			name: "GetAll_group1",
			args: args{
				gid:  dataG[1].ID,
				join: true,
			},
			wantPerm: []*db.Permission{
				dataP[2],
			},
		},
		{
			name: "GetAll_group2",
			args: args{
				gid:  dataG[2].ID,
				join: true,
			},
			wantPerm: []*db.Permission{
				dataP[3],
			},
		},
		{
			name: "GetAll_group_none",
			args: args{
				gid:  uuid.New(),
				join: true,
			},
			wantPerm: []*db.Permission{},
		},
		{
			name: "GetAll_group0_nojoin",
			args: args{
				gid: dataG[0].ID,
			},
			wantPerm: []*db.Permission{
				{
					GeneralFields: db.GeneralFields{
						ID: dataP[0].ID,
					},
					GID:  dataG[0].ID,
					PID:  dataPL[0].ID,
					Perm: db.PermAll,
				},
				{
					GeneralFields: db.GeneralFields{
						ID: dataP[1].ID,
					},
					GID:  dataG[0].ID,
					PID:  dataPL[1].ID,
					Perm: db.PermRead,
				},
			},
		},
		{
			name: "GetAll_user0",
			args: args{
				uid:  dataU[0].ID,
				join: true,
			},
			wantPerm: []*db.Permission{
				dataP[4],
			},
		},
		{
			name: "GetAll_user1",
			args: args{
				uid:  dataU[1].ID,
				join: true,
			},
			wantPerm: []*db.Permission{
				dataP[5],
				dataP[6],
			},
		},
		{
			name: "GetAll_user2",
			args: args{
				uid:  dataU[2].ID,
				join: true,
			},
			wantPerm: []*db.Permission{
				dataP[7],
			},
		},
		{
			name: "GetAll_user_none",
			args: args{
				uid:  uuid.New(),
				join: true,
			},
			wantPerm: []*db.Permission{},
		},
		{
			name: "GetAll_user1_nojoin",
			args: args{
				uid: dataU[1].ID,
			},
			wantPerm: []*db.Permission{
				{
					GeneralFields: db.GeneralFields{
						ID: dataP[5].ID,
					},
					UID:  dataU[1].ID,
					PID:  dataPL[0].ID,
					Perm: db.PermRead,
				},
				{
					GeneralFields: db.GeneralFields{
						ID: dataP[6].ID,
					},
					UID:  dataU[1].ID,
					PID:  dataPL[2].ID,
					Perm: db.PermWrite,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPerm, err := Permission.GetAll(tt.args.uid, tt.args.gid, tt.args.join)
			for _, v := range gotPerm {
				v.CreatedAt = 0
				v.UpdatedAt = 0
				if tt.args.join {
					v.PermissionList.CreatedAt = 0
					v.PermissionList.UpdatedAt = 0
				}
			}
			assert.Equal(t, err != nil, tt.wantErr)
			assert.ElementsMatch(t, gotPerm, tt.wantPerm)
		})
	}
}

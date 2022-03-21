package handler

import (
	"context"
	"io/ioutil"
	"os"
	"skynet/db"
	"skynet/sn"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
)

var (
	dataPL = []*sn.PermissionList{
		{
			GeneralFields: sn.GeneralFields{
				ID: uuid.New(),
			},
			Name: "perm1",
			Note: "p_abc",
		},
		{
			GeneralFields: sn.GeneralFields{
				ID: uuid.New(),
			},
			Name: "perm2",
			Note: "p_def",
		},
		{
			GeneralFields: sn.GeneralFields{
				ID: uuid.New(),
			},
			Name: "perm3",
			Note: "p_ghi",
		},
	}
	dataU = []*sn.User{
		{
			GeneralFields: sn.GeneralFields{
				ID: uuid.New(),
			},
			Username: "user1",
		},
		{
			GeneralFields: sn.GeneralFields{
				ID: uuid.New(),
			},
			Username: "user2",
		},
		{
			GeneralFields: sn.GeneralFields{
				ID: uuid.New(),
			},
			Username: "user3",
		},
	}
	dataG = []*sn.UserGroup{
		{
			GeneralFields: sn.GeneralFields{
				ID: uuid.New(),
			},
			Name: "group1",
			Note: "g_abc",
		},
		{
			GeneralFields: sn.GeneralFields{
				ID: uuid.New(),
			},
			Name: "group2",
			Note: "g_def",
		},
		{
			GeneralFields: sn.GeneralFields{
				ID: uuid.New(),
			},
			Name: "group3",
			Note: "g_ghi",
		},
	}
	dataUGL = []*sn.UserGroupLink{
		{
			UID: dataU[0].ID,
			GID: dataG[0].ID,
		},
		{
			UID: dataU[1].ID,
			GID: dataG[0].ID,
		},
	}

	dataP = []*sn.Permission{
		{
			GID:            dataG[0].ID,
			PID:            dataPL[0].ID,
			Perm:           sn.PermAll,
			PermissionList: dataPL[0],
		},
		{
			GID:            dataG[0].ID,
			PID:            dataPL[1].ID,
			Perm:           sn.PermRead,
			PermissionList: dataPL[1],
		},
		{
			GID:            dataG[1].ID,
			PID:            dataPL[1].ID,
			Perm:           sn.PermWrite,
			PermissionList: dataPL[1],
		},
		{
			GID:            dataG[2].ID,
			PID:            dataPL[2].ID,
			Perm:           sn.PermExecute,
			PermissionList: dataPL[2],
		},
		{
			UID:            dataU[0].ID,
			PID:            dataPL[0].ID,
			Perm:           sn.PermNone,
			PermissionList: dataPL[0],
		},
		{
			UID:            dataU[1].ID,
			PID:            dataPL[0].ID,
			Perm:           sn.PermRead,
			PermissionList: dataPL[0],
		},
		{
			UID:            dataU[1].ID,
			PID:            dataPL[2].ID,
			Perm:           sn.PermWrite,
			PermissionList: dataPL[2],
		},
		{
			UID:            dataU[2].ID,
			PID:            dataPL[1].ID,
			Perm:           sn.PermAll,
			PermissionList: dataPL[1],
		},
	}
)

func setup() *os.File {
	file, err := ioutil.TempFile("", "*.db")
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	sn.Skynet.DB = db.NewDB(ctx, &db.DBConfig{
		Type: db.DBType_Sqlite,
		Path: file.Name(),
	})
	sn.Skynet.Permission = NewPermission()
	if err := sn.Skynet.GetDB().Create(&dataPL).Error; err != nil {
		teardown(file)
		panic(err)
	}
	if err := sn.Skynet.GetDB().Omit("PermissionList").Create(&dataP).Error; err != nil {
		teardown(file)
		panic(err)
	}
	for _, v := range dataP {
		v.CreatedAt = 0
		v.UpdatedAt = 0
		v.PermissionList.CreatedAt = 0
		v.PermissionList.UpdatedAt = 0
	}
	sn.Skynet.GetDB().Logger = logger.Default.LogMode(logger.Info)
	return file
}

func teardown(f *os.File) {
	os.Remove(f.Name())
}

func TestMain(m *testing.M) {
	f := setup()
	code := m.Run()
	teardown(f)
	os.Exit(code)
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
		wantPerm []*sn.Permission
		wantErr  bool
	}{
		{
			name: "GetAll_group0",
			args: args{
				gid:  dataG[0].ID,
				join: true,
			},
			wantPerm: []*sn.Permission{
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
			wantPerm: []*sn.Permission{
				dataP[2],
			},
		},
		{
			name: "GetAll_group2",
			args: args{
				gid:  dataG[2].ID,
				join: true,
			},
			wantPerm: []*sn.Permission{
				dataP[3],
			},
		},
		{
			name: "GetAll_group_none",
			args: args{
				gid:  uuid.New(),
				join: true,
			},
			wantPerm: []*sn.Permission{},
		},
		{
			name: "GetAll_group0_nojoin",
			args: args{
				gid: dataG[0].ID,
			},
			wantPerm: []*sn.Permission{
				{
					GeneralFields: sn.GeneralFields{
						ID: dataP[0].ID,
					},
					GID:  dataG[0].ID,
					PID:  dataPL[0].ID,
					Perm: sn.PermAll,
				},
				{
					GeneralFields: sn.GeneralFields{
						ID: dataP[1].ID,
					},
					GID:  dataG[0].ID,
					PID:  dataPL[1].ID,
					Perm: sn.PermRead,
				},
			},
		},
		{
			name: "GetAll_user0",
			args: args{
				uid:  dataU[0].ID,
				join: true,
			},
			wantPerm: []*sn.Permission{
				dataP[4],
			},
		},
		{
			name: "GetAll_user1",
			args: args{
				uid:  dataU[1].ID,
				join: true,
			},
			wantPerm: []*sn.Permission{
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
			wantPerm: []*sn.Permission{
				dataP[7],
			},
		},
		{
			name: "GetAll_user_none",
			args: args{
				uid:  uuid.New(),
				join: true,
			},
			wantPerm: []*sn.Permission{},
		},
		{
			name: "GetAll_user1_nojoin",
			args: args{
				uid: dataU[1].ID,
			},
			wantPerm: []*sn.Permission{
				{
					GeneralFields: sn.GeneralFields{
						ID: dataP[5].ID,
					},
					UID:  dataU[1].ID,
					PID:  dataPL[0].ID,
					Perm: sn.PermRead,
				},
				{
					GeneralFields: sn.GeneralFields{
						ID: dataP[6].ID,
					},
					UID:  dataU[1].ID,
					PID:  dataPL[2].ID,
					Perm: sn.PermWrite,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPerm, err := sn.Skynet.Permission.GetAll(tt.args.uid, tt.args.gid, tt.args.join)
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

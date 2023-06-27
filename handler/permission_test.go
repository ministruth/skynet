package handler

import (
	"flag"
	"os"
	"testing"

	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/sn"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
)

var (
	dataP   = make(map[int]*sn.Permission)
	dataU   = make(map[int]*sn.User)
	dataG   = make(map[int]*sn.Group)
	dataUGL = make(map[int]*sn.UserGroupLink)
	dataPL  = make(map[int]*sn.PermissionLink)
)

func ToSlice[T any](m map[int]T) []T {
	ret := make([]T, 0, len(m))
	for _, v := range m {
		ret = append(ret, v)
	}
	return ret
}

func G2PE(g *sn.Group, p *sn.PermissionLink) *sn.PermEntry {
	return &sn.PermEntry{
		ID:        g.ID,
		Name:      g.Name,
		Note:      g.Note,
		Perm:      p.Perm,
		Origin:    nil,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func AddP(i int, name string, note string) {
	dataP[i] = &sn.Permission{
		GeneralFields: sn.GeneralFields{ID: uuid.New()},
		Name:          name,
		Note:          note,
	}
}

func AddU(i int, name string) {
	dataU[i] = &sn.User{
		GeneralFields: sn.GeneralFields{ID: uuid.New()},
		Username:      name,
		Password:      "123",
	}
}

func AddG(i int, name string, note string) {
	dataG[i] = &sn.Group{
		GeneralFields: sn.GeneralFields{ID: uuid.New()},
		Name:          name,
		Note:          note,
	}
}

func AddUGL(i int, ui int, gi int) {
	dataUGL[i] = &sn.UserGroupLink{
		GeneralFields: sn.GeneralFields{ID: uuid.New()},
		UID:           dataU[ui].ID,
		GID:           dataG[gi].ID,
	}
}

func AddPL(i int, ui int, gi int, pi int, perm int32) {
	var uid, gid uuid.NullUUID
	// var uid, gid, pid uuid.UUID
	if ui != 0 {
		uid = uuid.NullUUID{UUID: dataU[ui].ID, Valid: true}
	}
	if gi != 0 {
		gid = uuid.NullUUID{UUID: dataG[gi].ID, Valid: true}
	}
	dataPL[i] = &sn.PermissionLink{
		GeneralFields: sn.GeneralFields{ID: uuid.New()},
		UID:           uid,
		GID:           gid,
		PID:           dataP[pi].ID,
		Perm:          perm,
		Permission:    dataP[pi],
	}
	if ui != 0 {
		dataPL[i].User = dataU[ui]
	}
	if gi != 0 {
		dataPL[i].Group = dataG[gi]
	}
}

func setup() {
	flag.Parse()
	var err error
	verbose := flag.Lookup("test.v").Value.String() == "true"
	// sn.Skynet.DB, err = db.NewDB("debug.db", "sqlite", verbose)
	sn.Skynet.DB, err = db.NewDB("file::memory:?cache=shared", "sqlite", verbose)
	if err != nil {
		panic(err)
	}
	Init()

	AddP(1, "perm1", "perm_123")
	AddP(2, "perm2", "perm_456")
	AddP(3, "perm3", "perm_789")
	AddG(1, "group1", "group_123")
	AddG(2, "group2", "group_456")
	AddG(3, "group3", "group_789")
	AddU(1, "user1")
	AddU(2, "user2")
	AddU(3, "user3")
	AddUGL(1, 2, 1)
	AddUGL(2, 2, 2)
	AddUGL(3, 2, 3)
	AddUGL(4, 3, 1)
	AddUGL(5, 3, 2)
	AddPL(1, 0, 1, 1, sn.PermAll)
	AddPL(2, 0, 1, 2, sn.PermRead)
	AddPL(3, 0, 1, 3, sn.PermWriteExecute)
	AddPL(4, 0, 2, 1, sn.PermRead)
	AddPL(5, 0, 2, 2, sn.PermWrite)
	AddPL(6, 1, 0, 1, sn.PermAll)
	AddPL(7, 1, 0, 2, sn.PermRead)
	AddPL(8, 1, 0, 3, sn.PermWriteExecute)
	AddPL(9, 2, 0, 1, sn.PermWrite)
	AddPL(10, 3, 0, 2, sn.PermRead)

	tmp0 := ToSlice(dataU)
	if err := sn.Skynet.DB.Create(&tmp0).Error; err != nil {
		panic(err)
	}
	tmp1 := ToSlice(dataG)
	if err := sn.Skynet.DB.Create(&tmp1).Error; err != nil {
		panic(err)
	}
	tmp2 := ToSlice(dataP)
	if err := sn.Skynet.DB.Create(&tmp2).Error; err != nil {
		panic(err)
	}
	tmp3 := ToSlice(dataPL)
	if err := sn.Skynet.DB.Omit("User", "Group", "Permission").Create(&tmp3).Error; err != nil {
		panic(err)
	}
	tmp4 := ToSlice(dataUGL)
	if err := sn.Skynet.DB.Create(&tmp4).Error; err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func Test_Permission_GetUserMerged(t *testing.T) {
	type args struct {
		id uuid.UUID
	}
	tests := []struct {
		name     string
		args     args
		wantPerm map[uuid.UUID]*sn.PermEntry
		wantErr  bool
	}{
		{
			name: "GetUserMerged_user1",
			args: args{
				id: dataU[1].ID,
			},
			wantPerm: map[uuid.UUID]*sn.PermEntry{
				dataP[1].ID: {
					ID:     dataP[1].ID,
					Name:   dataP[1].Name,
					Perm:   sn.PermAll,
					Origin: nil,
				},
				dataP[2].ID: {
					ID:     dataP[2].ID,
					Name:   dataP[2].Name,
					Perm:   sn.PermRead,
					Origin: nil,
				},
				dataP[3].ID: {
					ID:     dataP[3].ID,
					Name:   dataP[3].Name,
					Perm:   sn.PermWriteExecute,
					Origin: nil,
				}},
			wantErr: false,
		},
		{
			name: "GetUserMerged_user2",
			args: args{
				id: dataU[2].ID,
			},
			wantPerm: map[uuid.UUID]*sn.PermEntry{
				dataP[1].ID: {
					ID:     dataP[1].ID,
					Name:   dataP[1].Name,
					Perm:   sn.PermWrite,
					Origin: nil,
				},
				dataP[2].ID: {
					ID:     dataP[2].ID,
					Name:   dataP[2].Name,
					Perm:   sn.PermRead | sn.PermWrite,
					Origin: []*sn.PermEntry{G2PE(dataG[1], dataPL[2]), G2PE(dataG[2], dataPL[5])},
				},
				dataP[3].ID: {
					ID:     dataP[3].ID,
					Name:   dataP[3].Name,
					Perm:   sn.PermWriteExecute,
					Origin: []*sn.PermEntry{G2PE(dataG[1], dataPL[3])},
				}},
			wantErr: false,
		},
		{
			name: "GetUserMerged_user3",
			args: args{
				id: dataU[3].ID,
			},
			wantPerm: map[uuid.UUID]*sn.PermEntry{
				dataP[1].ID: {
					ID:     dataP[1].ID,
					Name:   dataP[1].Name,
					Perm:   sn.PermAll,
					Origin: []*sn.PermEntry{G2PE(dataG[2], dataPL[4]), G2PE(dataG[1], dataPL[1])},
				},
				dataP[2].ID: {
					ID:     dataP[2].ID,
					Name:   dataP[2].Name,
					Perm:   sn.PermRead,
					Origin: nil,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPerm, err := sn.Skynet.Permission.GetUserMerged(tt.args.id)
			assert.Equal(t, tt.wantErr, err != nil)
			for k, v := range tt.wantPerm {
				assert.Contains(t, gotPerm, k)
				assert.Equal(t, v.ID, gotPerm[k].ID)
				assert.Equal(t, v.Name, gotPerm[k].Name)
				assert.Equal(t, v.Perm, gotPerm[k].Perm)
				assert.ElementsMatch(t, v.Origin, gotPerm[k].Origin)
			}
		})
	}
}

func Test_Permission_GetAll(t *testing.T) {
	type args struct {
		uid       uuid.UUID
		gid       uuid.UUID
		joinUser  bool
		joinGroup bool
		joinPerm  bool
	}
	tests := []struct {
		name     string
		args     args
		wantPerm []*sn.PermissionLink
		wantErr  bool
	}{
		{
			name: "GetAll_group1",
			args: args{
				gid:       dataG[1].ID,
				joinUser:  false,
				joinGroup: false,
				joinPerm:  false,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[1],
				dataPL[2],
				dataPL[3],
			},
		},
		{
			name: "GetAll_group1_join1",
			args: args{
				gid:       dataG[1].ID,
				joinUser:  false,
				joinGroup: false,
				joinPerm:  true,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[1],
				dataPL[2],
				dataPL[3],
			},
		},
		{
			name: "GetAll_group1_join2",
			args: args{
				gid:       dataG[1].ID,
				joinUser:  false,
				joinGroup: true,
				joinPerm:  false,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[1],
				dataPL[2],
				dataPL[3],
			},
		},
		{
			name: "GetAll_group1_join3",
			args: args{
				gid:       dataG[1].ID,
				joinUser:  false,
				joinGroup: true,
				joinPerm:  true,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[1],
				dataPL[2],
				dataPL[3],
			},
		},
		{
			name: "GetAll_group2",
			args: args{
				gid:       dataG[2].ID,
				joinUser:  false,
				joinGroup: false,
				joinPerm:  false,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[4],
				dataPL[5],
			},
		},
		{
			name: "GetAll_group3",
			args: args{
				gid:       dataG[3].ID,
				joinUser:  false,
				joinGroup: false,
				joinPerm:  false,
			},
			wantPerm: []*sn.PermissionLink{},
		},
		{
			name: "GetAll_user1",
			args: args{
				uid:       dataU[1].ID,
				joinUser:  false,
				joinGroup: false,
				joinPerm:  false,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[6],
				dataPL[7],
				dataPL[8],
			},
		},
		{
			name: "GetAll_user1_join1",
			args: args{
				uid:       dataU[1].ID,
				joinUser:  false,
				joinGroup: false,
				joinPerm:  true,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[6],
				dataPL[7],
				dataPL[8],
			},
		},
		{
			name: "GetAll_user1_join2",
			args: args{
				uid:       dataU[1].ID,
				joinUser:  true,
				joinGroup: false,
				joinPerm:  false,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[6],
				dataPL[7],
				dataPL[8],
			},
		},
		{
			name: "GetAll_user2",
			args: args{
				uid:       dataU[2].ID,
				joinUser:  false,
				joinGroup: false,
				joinPerm:  false,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[9],
			},
		},
		{
			name: "GetAll_user3",
			args: args{
				uid:       dataU[3].ID,
				joinUser:  false,
				joinGroup: false,
				joinPerm:  false,
			},
			wantPerm: []*sn.PermissionLink{
				dataPL[10],
			},
		},
		{
			name: "GetAll_nil1",
			args: args{
				uid:       dataU[1].ID,
				gid:       dataG[1].ID,
				joinUser:  false,
				joinGroup: false,
				joinPerm:  false,
			},
			wantPerm: nil,
		},
		{
			name: "GetAll_nil2",
			args: args{
				joinUser:  false,
				joinGroup: false,
				joinPerm:  false,
			},
			wantPerm: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPerm, err := sn.Skynet.Permission.GetAll(tt.args.uid, tt.args.gid,
				tt.args.joinUser, tt.args.joinGroup, tt.args.joinPerm)
			var want []*sn.PermissionLink
			copier.Copy(&want, &tt.wantPerm)
			if !tt.args.joinUser {
				for _, v := range want {
					v.User = nil
				}
			}
			if !tt.args.joinGroup {
				for _, v := range want {
					v.Group = nil
				}
			}
			if !tt.args.joinPerm {
				for _, v := range want {
					v.Permission = nil
				}
			}
			assert.Equal(t, tt.wantErr, err != nil)
			assert.ElementsMatch(t, want, gotPerm)
		})
	}
}

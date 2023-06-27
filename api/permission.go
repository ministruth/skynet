package api

import "github.com/MXWXZ/skynet/sn"

func APIGetPermission(req *sn.Request) (*sn.Response, error) {
	perm, err := sn.Skynet.Permission.GetEntry()
	if err != nil {
		return nil, err
	}
	ret := []*sn.Permission{}
	for _, v := range perm {
		if v.ID == sn.Skynet.ID.Get(sn.PermGuestID) || v.ID == sn.Skynet.ID.Get(sn.PermUserID) {
			continue
		}
		ret = append(ret, v)
	}
	return &sn.Response{Data: ret}, nil
}

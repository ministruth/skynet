package sn

import (
	"skynet/sn/tpl"

	"github.com/google/uuid"
)

type UserPerm int32

const (
	PermNone    UserPerm = 0        // PermNone is default, no permission
	PermExecute UserPerm = 1        // PermExecute can execute
	PermWrite   UserPerm = 1 << 1   // PermWrite can write database
	PermRead    UserPerm = 1 << 2   // PermRead can read
	PermAll     UserPerm = 1<<3 - 1 // PermAll give all permission

	PermWriteExecute UserPerm = PermWrite | PermExecute // PermWriteExecute can write and execute
)

type SNPerm struct {
	ID   uuid.UUID
	Name string // filled automatically
	Perm UserPerm
}

type SNPermListType int32

const (
	PermListUser SNPermListType = iota
	PermListGroup
)

type SNPermList struct {
	Perm  tpl.SafeMap[uuid.UUID, *SNPerm]
	Group []uuid.UUID
	Type  SNPermListType
}

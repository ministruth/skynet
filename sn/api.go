package sn

import (
	"github.com/gin-gonic/gin"
)

type SNAPI interface {
	GetRouter() *gin.RouterGroup
	AddAPIItem(i []*SNAPIItem)
}

type SNAPIMethod int

const (
	APIGet SNAPIMethod = iota
	APIPost
	APIPut
	APIPatch
	APIDelete
	APIOptions
	APIHead
	APIAny
)

type SNAPIFunc func(c *gin.Context, u *User) (int, error)

type SNAPIItem struct {
	Path   string
	Method SNAPIMethod
	Role   UserRole
	Func   SNAPIFunc
}

package sn

import (
	"github.com/gin-gonic/gin"
)

// SNAPI is interface for skynet API.
type SNAPI interface {
	// GetRouter returns using gin routergroup.
	GetRouter() *gin.RouterGroup

	// GetAPI returns current API list.
	GetAPI() []*SNAPIItem

	// AddAPI adds item to API list.
	AddAPI(i []*SNAPIItem)
}

// SNAPIMethod is http method for API.
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

// SNAPIFunc is API function type.
//
// u is current user when user signin, otherwise nil.
//
// return http code and error, code is only used when error not nil.
//	// these are same
//	return 0, nil
//	return 200, nil
//	return 500, nil
type SNAPIFunc func(c *gin.Context, u *User) (int, error)

// SNAPIItem is APIItem struct.
type SNAPIItem struct {
	Path   string      // API path
	Method SNAPIMethod // API http method
	Role   UserRole    // API allow role
	Func   SNAPIFunc   // API function
}

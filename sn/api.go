package sn

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SNAPI is interface for skynet API.
type SNAPI interface {
	// GetRouter returns using gin routergroup.
	GetRouter() *gin.RouterGroup

	// GetAPI returns current API list.
	GetAPI() []*SNAPIItem

	// AddAPI adds item to API list.
	AddAPI(item []*SNAPIItem)

	// GetMenu returns current menu.
	GetMenu() []*SNMenu

	// AddMenu add menu item to parent.
	//
	// If parent not exist, return false.
	AddMenu(item *SNMenu, parent uuid.UUID) bool
}

// SNAPIMethod is http method for API.
type SNAPIMethod int32

const (
	// APIGet represents HTTP Get method
	APIGet SNAPIMethod = iota
	// APIPost represents HTTP Post method
	APIPost
	// APIPut represents HTTP Put method
	APIPut
	// APIPatch represents HTTP Patch method
	APIPatch
	// APIDelete represents HTTP Delete method
	APIDelete
	// APIOptions represents HTTP Options method
	APIOptions
	// APIHead represents HTTP Head method
	APIHead
	// APIAny represents any HTTP method
	APIAny
)

// SNAPIFunc is API function type.
//
// id is current user id when user signin, otherwise uuid.Nil.
//
// return http code and error, when error is nil and code not zero,
// skynet will handle it based on the code
type SNAPIFunc func(c *gin.Context, id uuid.UUID) (int, error)

// SNCheckerFunc is permission checker function type.
type SNCheckerFunc func(perm map[uuid.UUID]*SNPerm) bool

// SNAPIItem is APIItem struct.
type SNAPIItem struct {
	Path    string      // API path
	Method  SNAPIMethod // API http method
	Perm    *SNPerm     // API permission
	Func    SNAPIFunc   // API function
	Checker SNCheckerFunc
}

type SNMenu struct {
	ID       uuid.UUID
	Name     string
	Path     string
	Icon     string
	Children []*SNMenu
	Perm     *SNPerm
	Checker  SNCheckerFunc
}

package sn

type DefaultID int32

const (
	GroupRootID DefaultID = iota // root user group
	PermAllID                    // full permission
	PermUserID                   // login user permission
	PermGuestID                  // guest permission

	PermManageUserID         // manage user
	PermManageUserPermID     // manage user permission
	PermManageGroupID        // manage user group
	PermManageGroupPermID    // manage user group
	PermManageNotificationID // manage notification
	PermManageSystemID       // manage system
	PermManagePluginID       // manage plugin

	DefaultIDMax // max id count
)

// SNDB is interface for db.
type SNDB[T any] interface {
	// Get return db realization object, exit when facing any error.
	Get() T
}

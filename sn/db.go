package sn

// SNDB is interface for db.
type SNDB interface {
	// Get return db realization object, exit when facing any error.
	Get() interface{}
}

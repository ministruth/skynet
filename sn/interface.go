package sn

import (
	"github.com/google/uuid"
)

type ReCAPTCHA interface {
	Verify(response string, ip string) error
}

type Menu interface {
	GetAll() []*MenuItem
	GetByID(id uuid.UUID) *MenuItem
	Add(item *MenuItem, parent uuid.UUID) bool
}

type PermEntry struct {
	ID        uuid.UUID
	Name      string // filled automatically
	Note      string // filled automatically
	Perm      UserPerm
	Origin    []*PermEntry
	CreatedAt int64 `json:"created_at"` // filled automatically
	UpdatedAt int64 `json:"updated_at"` // filled automatically
}

// Check check PermEntry with permission map.
func (p *PermEntry) Check(perm map[uuid.UUID]*PermEntry) bool {
	if p.ID == Skynet.ID.Get(PermGuestID) {
		return true
	}
	if perm != nil {
		if e, ok := perm[p.ID]; ok { // user permission check first
			return (e.Perm & p.Perm) == p.Perm
		}
		if e, ok := perm[Skynet.ID.Get(PermAllID)]; ok {
			return (e.Perm & p.Perm) == p.Perm
		}
	}
	return false // fail safe
}

// SNCheckerFunc is permission checker function type.
type PermCheckerFunc func(perm map[uuid.UUID]*PermEntry) bool

type MenuBadgeFunc func() int64

type MenuItem struct {
	ID        uuid.UUID
	Name      string
	Path      string
	Icon      string
	BadgeFunc MenuBadgeFunc
	OmitEmpty bool
	Children  []*MenuItem
	Perm      *PermEntry
	Checker   PermCheckerFunc
}

func (m *MenuItem) Check(p map[uuid.UUID]*PermEntry) bool {
	if m.Checker == nil && m.Perm == nil { // menu group
		return true
	} else {
		if m.Checker != nil {
			return m.Checker(p)
		}
		ok := false
		if m.Perm != nil {
			ok = m.Perm.Check(p)
		}
		return ok
	}
}

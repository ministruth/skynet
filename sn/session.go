package sn

import (
	"crypto/rand"
	"encoding/base32"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/rbcervilla/redisstore/v9"
	"github.com/ztrue/tracerr"
)

// ErrSessionInvalid raises when session is invalid.
var ErrSessionInvalid = tracerr.New("session invalid")

// SessionData is the structure stored in session.
type SessionData struct {
	ID   uuid.UUID `gob:"id"`
	Time int64     `gob:"time"`
}

// generateRandomKey generate random session key.
func (s *SessionData) generateRandomKey() string {
	k := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		panic(err)
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(k), "=")
}

// SaveSession saves SessionData to sessions.Session.
func (s *SessionData) SaveSession(session *sessions.Session) {
	session.Values["id"] = s.ID
	session.Values["time"] = s.Time
	session.ID = strings.ToUpper(s.ID.String() + "_" + s.generateRandomKey())
}

// Session defines interface for any session implementation.
type Session interface {
	// GetStore returns inner redis store.
	GetStore() *redisstore.RedisStore

	// Find finds all sessions associate to user by uid.
	// When uid is nil, find all sessions.
	Find(uid []uuid.UUID) (map[string][]*SessionData, error)

	// Delete deletes all sessions associate to user by uid.
	// When uid is nil, delete all sessions.
	Delete(uid []uuid.UUID) error
}

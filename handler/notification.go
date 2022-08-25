package handler

import (
	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/utils"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type NotificationImpl struct {
	orm *db.ORM[db.Notification]
}

var Notification = &NotificationImpl{}

func (p *NotificationImpl) WithTx(tx *gorm.DB) *NotificationImpl {
	return &NotificationImpl{
		orm: db.NewORM[db.Notification](tx),
	}
}

func (s *NotificationImpl) New(level db.NotifyLevel, name string, message string, detail string) error {
	return s.orm.Create(&db.Notification{
		Level:   level,
		Name:    name,
		Message: message,
		Detail:  detail,
	})
}

// GetAll get all notification by condition.
func (u *NotificationImpl) GetAll(cond *db.Condition) ([]*db.Notification, error) {
	return u.orm.Cond(cond).Find()
}

// Get get notification by id.
func (u *NotificationImpl) Get(id uuid.UUID) (*db.Notification, error) {
	return u.orm.Take(id)
}

// Count count notification by condition.
func (u *NotificationImpl) Count(cond *db.Condition) (int64, error) {
	return u.orm.Count(cond)
}

// Delete delete notification by id.
func (u *NotificationImpl) Delete(id uuid.UUID) (bool, error) {
	return u.orm.DeleteID(id)
}

// Delete delete all notification.
func (u *NotificationImpl) DeleteAll() (int64, error) {
	return u.orm.DeleteAll()
}

type NotificationHook struct{}

func (h NotificationHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
	}
}

func (h NotificationHook) Fire(e *logrus.Entry) error {
	var level db.NotifyLevel
	switch e.Level {
	case logrus.WarnLevel:
		level = db.NotifyWarning
	case logrus.ErrorLevel:
		level = db.NotifyError
	case logrus.FatalLevel:
		level = db.NotifyFatal
	}
	// log may in transaction, prevent deadlock
	go Notification.New(level, "Skynet log", e.Message, utils.MustMarshal(e.Data))
	return nil
}

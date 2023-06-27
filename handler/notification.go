package handler

import (
	"sync/atomic"

	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type NotificationImpl struct {
	orm    *sn.ORM[sn.Notification]
	unread *int64
}

func NewNotificationHandler() sn.NotificationHandler {
	var unread int64
	return &NotificationImpl{orm: sn.NewORM[sn.Notification](sn.Skynet.DB), unread: &unread}
}

func (impl *NotificationImpl) WithTx(tx *gorm.DB) sn.NotificationHandler {
	return &NotificationImpl{
		orm:    sn.NewORM[sn.Notification](tx),
		unread: impl.unread,
	}
}

func (impl *NotificationImpl) SetUnread(num int64) {
	atomic.StoreInt64(impl.unread, num)
}

func (impl *NotificationImpl) GetUnread() int64 {
	return atomic.LoadInt64(impl.unread)
}

func (impl *NotificationImpl) New(level sn.NotifyLevel, name string, message string, detail string) error {
	err := impl.orm.Create(&sn.Notification{
		Level:   level,
		Name:    name,
		Message: message,
		Detail:  detail,
	})
	if level > sn.NotifySuccess && err == nil {
		atomic.AddInt64(impl.unread, 1)
	}
	return err
}

func (impl *NotificationImpl) GetAll(cond *sn.Condition) ([]*sn.Notification, error) {
	return impl.orm.Cond(cond).Find()
}

func (impl *NotificationImpl) Get(id uuid.UUID) (*sn.Notification, error) {
	return impl.orm.Take(id)
}

func (impl *NotificationImpl) Count(cond *sn.Condition) (int64, error) {
	return impl.orm.Count(cond)
}

func (impl *NotificationImpl) Delete(id uuid.UUID) (bool, error) {
	return impl.orm.DeleteID(id)
}

func (impl *NotificationImpl) DeleteAll() (int64, error) {
	return impl.orm.DeleteAll()
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
	var level sn.NotifyLevel
	switch e.Level {
	case logrus.WarnLevel:
		level = sn.NotifyWarning
	case logrus.ErrorLevel:
		level = sn.NotifyError
	case logrus.FatalLevel:
		level = sn.NotifyFatal
	}
	// log may in transaction, prevent deadlock
	go sn.Skynet.Notification.New(level, "Skynet log", e.Message, utils.MustMarshal(e.Data))
	return nil
}

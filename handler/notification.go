package handler

import (
	"skynet/sn"
	"skynet/sn/impl"
	"skynet/sn/utils"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type siteNotification struct {
	*impl.ORM[sn.Notification]
}

func NewNotification() sn.SNNotification {
	return &siteNotification{
		ORM: impl.NewORM[sn.Notification](nil),
	}
}

func (p *siteNotification) WithTx(tx *gorm.DB) sn.SNNotification {
	return &siteNotification{
		ORM: impl.NewORM[sn.Notification](tx),
	}
}

func (s *siteNotification) New(level sn.NotifyLevel, name string, message string, detail string) error {
	return s.Impl.Create(&sn.Notification{
		Level:   level,
		Name:    name,
		Message: message,
		Detail:  detail,
	})
}

type NotificationHook struct{}

func (h NotificationHook) Levels() []log.Level {
	return []log.Level{
		log.WarnLevel,
		log.ErrorLevel,
		log.FatalLevel,
	}
}

func (h NotificationHook) Fire(e *log.Entry) error {
	var level sn.NotifyLevel
	switch e.Level {
	case log.WarnLevel:
		level = sn.NotifyWarning
	case log.ErrorLevel:
		level = sn.NotifyError
	case log.FatalLevel:
		level = sn.NotifyFatal
	}
	// log may in transaction, prevent deadlock
	go sn.Skynet.Notification.New(level, "Skynet log", e.Message, utils.MustMarshal(e.Data))
	return nil
}

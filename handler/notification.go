package handler

import (
	"fmt"
	"skynet/sn"
	"skynet/sn/utils"

	log "github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

type siteNotification struct{}

func NewNotification() sn.SNNotification {
	return &siteNotification{}
}

func (s *siteNotification) New(level sn.NotifyLevel, name string, message string) error {
	notify := sn.Notification{
		Level:   level,
		Name:    name,
		Message: message,
	}
	return tracerr.Wrap(utils.GetDB().Create(&notify).Error)
}

func (s *siteNotification) Delete(id int) error {
	if id == 0 {
		return s.DeleteAll()
	} else {
		return tracerr.Wrap(utils.GetDB().Delete(&sn.Notification{}, id).Error)
	}
}

func (s *siteNotification) DeleteAll() error {
	return tracerr.Wrap(utils.GetDB().Where("1 = 1").Delete(&sn.Notification{}).Error)
}

func (s *siteNotification) MarkRead(id int) error {
	if id == 0 {
		return s.MarkAllRead()
	} else {
		return tracerr.Wrap(utils.GetDB().Model(&sn.Notification{}).Where("id = ?", id).Update("read", 1).Error)
	}
}

func (s *siteNotification) MarkAllRead() error {
	return tracerr.Wrap(utils.GetDB().Model(&sn.Notification{}).Where("read = ?", "0").Update("read", 1).Error)
}

func (s *siteNotification) Count(read interface{}) (int64, error) {
	var count int64
	var err error
	if read == nil {
		err = tracerr.Wrap(utils.GetDB().Model(&sn.Notification{}).Count(&count).Error)
	} else if read.(bool) {
		err = tracerr.Wrap(utils.GetDB().Model(&sn.Notification{}).Where("read = ?", 1).Count(&count).Error)
	} else {
		err = tracerr.Wrap(utils.GetDB().Model(&sn.Notification{}).Where("read = ?", 0).Count(&count).Error)
	}
	return count, err
}

func (s *siteNotification) GetByID(id int) (*sn.Notification, error) {
	var ret sn.Notification
	if err := tracerr.Wrap(utils.GetDB().First(&ret, id).Error); err != nil {
		return nil, err
	}
	return &ret, nil
}

func (s *siteNotification) GetAll(cond *sn.SNCondition) ([]*sn.Notification, error) {
	var ret []*sn.Notification
	return ret, tracerr.Wrap(utils.DBParseCondition(cond).Find(&ret).Error)
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
	var msg string
	for k, v := range e.Data {
		if k != "caller" && k != "stack" && k != "debug" {
			msg = fmt.Sprintf("%v %v:%v", msg, k, v)
		}
	}
	return sn.Skynet.Notification.New(level, "Skynet log", e.Message+msg)
}

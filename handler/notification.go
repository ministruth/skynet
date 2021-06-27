package handler

import (
	"skynet/sn"
	"skynet/sn/utils"

	log "github.com/sirupsen/logrus"
)

type siteNotification struct{}

func NewNotification() sn.SNNotification {
	return &siteNotification{}
}

func (s *siteNotification) New(level sn.NotifyLevel, name string, message string) error {
	notify := sn.Notifications{
		Level:   level,
		Name:    name,
		Message: message,
	}
	return utils.GetDB().Create(&notify).Error
}

func (s *siteNotification) Delete(id int) error {
	if id == 0 {
		return s.DeleteAll()
	} else {
		return utils.GetDB().Delete(&sn.Notifications{}, id).Error
	}
}

func (s *siteNotification) DeleteAll() error {
	return utils.GetDB().Where("1 = 1").Delete(&sn.Notifications{}).Error
}

func (s *siteNotification) MarkRead(id int) error {
	if id == 0 {
		return s.MarkAllRead()
	} else {
		return utils.GetDB().Model(&sn.Notifications{}).Where("id = ?", id).Update("read", 1).Error
	}
}

func (s *siteNotification) MarkAllRead() error {
	return utils.GetDB().Model(&sn.Notifications{}).Where("read = ?", "0").Update("read", 1).Error
}

func (s *siteNotification) Count(read interface{}) (int64, error) {
	var count int64
	var err error
	if read == nil {
		err = utils.GetDB().Model(&sn.Notifications{}).Count(&count).Error
	} else if read.(bool) {
		err = utils.GetDB().Model(&sn.Notifications{}).Where("read = ?", 1).Count(&count).Error
	} else {
		err = utils.GetDB().Model(&sn.Notifications{}).Where("read = ?", 0).Count(&count).Error
	}
	return count, err
}

func (s *siteNotification) GetByID(id int) (*sn.Notifications, error) {
	var ret sn.Notifications
	err := utils.GetDB().First(&ret, id).Error
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (s *siteNotification) GetAll(cond *sn.SNCondition) ([]*sn.Notifications, error) {
	var ret []*sn.Notifications
	return ret, utils.DBParseCondition(cond).Find(&ret).Error
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
	return sn.Skynet.Notification.New(level, "Skynet log", e.Message)
}
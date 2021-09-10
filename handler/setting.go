package handler

import (
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/ztrue/tracerr"
)

type siteSetting struct {
	setting *utils.StringMap
}

func NewSetting() (sn.SNSetting, error) {
	var ret siteSetting
	ret.setting = &utils.StringMap{}
	var rec []sn.Setting
	if err := tracerr.Wrap(utils.GetDB().Find(&rec).Error); err != nil {
		return nil, err
	}
	for _, v := range rec {
		ret.setting.Set(v.Name, v.Value)
	}
	return &ret, nil
}

func (s *siteSetting) GetAll(cond *sn.SNCondition) ([]*sn.Setting, error) {
	var ret []*sn.Setting
	return ret, tracerr.Wrap(utils.DBParseCondition(cond).Find(&ret).Error)
}

func (s *siteSetting) GetCache() map[string]interface{} {
	return s.setting.Map()
}

func (s *siteSetting) Get(name string) (string, bool) {
	ret, exist := s.setting.Get(name)
	if !exist {
		return "", exist
	}
	return ret.(string), exist
}

func (s *siteSetting) Set(name string, value string) error {
	v, exist := s.setting.Get(name)
	if !exist {
		err := tracerr.Wrap(utils.GetDB().Create(&sn.Setting{
			Name:  name,
			Value: value,
		}).Error)
		if err != nil {
			return err
		}
		s.setting.Set(name, value)
		return nil
	} else {
		if v != value {
			s.setting.Set(name, value)
			err := tracerr.Wrap(utils.GetDB().Model(&sn.Setting{}).Where("name = ?", name).Update("value", value).Error)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *siteSetting) Delete(name string) error {
	if _, exist := s.setting.Get(name); exist {
		s.setting.Delete(name)
		err := tracerr.Wrap(utils.GetDB().Where("name = ?", name).Delete(&sn.Setting{}).Error)
		if err != nil {
			return err
		}
	}
	return nil
}

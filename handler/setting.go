package handler

import (
	"skynet/sn"
	"skynet/sn/utils"
)

type siteSetting struct {
	setting map[string]string
}

func NewSetting() (sn.SNSetting, error) {
	var ret siteSetting
	ret.setting = make(map[string]string)

	var rec []sn.Setting
	err := utils.GetDB().Find(&rec).Error
	if err != nil {
		return nil, err
	}
	for _, v := range rec {
		ret.setting[v.Name] = v.Value
	}
	return &ret, nil
}

func (s *siteSetting) GetAll(cond *sn.SNCondition) ([]*sn.Setting, error) {
	var ret []*sn.Setting
	return ret, utils.DBParseCondition(cond).Find(&ret).Error
}

func (s *siteSetting) GetCache() map[string]string {
	return s.setting
}

func (s *siteSetting) Get(name string) (string, bool) {
	ret, exist := s.setting[name]
	return ret, exist
}

func (s *siteSetting) New(name string, value string) error {
	if i, exist := s.setting[name]; !exist || i != value {
		err := utils.GetDB().Create(&sn.Setting{
			Name:  name,
			Value: value,
		}).Error
		if err != nil {
			return err
		}
		s.setting[name] = value
	}
	return nil
}

func (s *siteSetting) Update(name string, value string) error {
	v, exist := s.setting[name]
	if !exist {
		return s.New(name, value)
	} else {
		if v != value {
			s.setting[name] = value
			err := utils.GetDB().Model(&sn.Setting{}).Where("name = ?", name).Update("value", value).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *siteSetting) Delete(name string) error {
	if _, exist := s.setting[name]; exist {
		delete(s.setting, name)
		err := utils.GetDB().Where("name = ?", name).Delete(&sn.Setting{}).Error
		if err != nil {
			return err
		}
	}
	return nil
}

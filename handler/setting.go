package handler

import (
	"skynet/sn"
	"skynet/sn/impl"
	"skynet/sn/tpl"

	"gorm.io/gorm"
)

type siteSetting struct {
	*impl.ORM[sn.Setting]
	cache *tpl.SafeMap[string, string]
}

func NewSetting() (sn.SNSetting, error) {
	ret := siteSetting{
		ORM:   impl.NewORM[sn.Setting](nil),
		cache: new(tpl.SafeMap[string, string]),
	}
	if err := ret.BuildCache(); err != nil {
		return nil, err
	}
	return &ret, nil
}

func (p *siteSetting) WithTx(tx *gorm.DB) sn.SNSetting {
	return &siteSetting{
		ORM:   impl.NewORM[sn.Setting](tx),
		cache: p.cache,
	}
}

func (s *siteSetting) BuildCache() error {
	rec, err := s.Impl.Find()
	if err != nil {
		return err
	}
	for _, v := range rec {
		s.cache.Set(v.Name, v.Value)
	}
	return nil
}

func (s *siteSetting) GetAll() map[string]string {
	return s.cache.Map()
}

func (s *siteSetting) Get(name string) (string, bool) {
	return s.cache.Get(name)
}

func (s *siteSetting) Set(name string, value string) error {
	v, ok := s.cache.Get(name)
	if !ok {
		if err := s.Impl.Create(&sn.Setting{
			Name:  name,
			Value: value,
		}); err != nil {
			return err
		}
		s.cache.Set(name, value)
		return nil
	} else {
		if v != value {
			if err := s.Impl.Where("name = ?", name).Update("value", value); err != nil {
				return err
			}
			s.cache.Set(name, value)
		}
	}
	return nil
}

func (s *siteSetting) Delete(name string) (bool, error) {
	if s.cache.Has(name) {
		row, err := s.Impl.Where("name = ?", name).Delete()
		if err != nil {
			return false, err
		}
		s.cache.Delete(name)
		return row == 1, nil
	}
	return false, nil
}

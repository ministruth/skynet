package handler

import (
	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/utils/tpl"

	"gorm.io/gorm"
)

type SettingImpl struct {
	orm   *db.ORM[db.Setting]
	cache *tpl.SafeMap[string, string]
}

var Setting = &SettingImpl{
	cache: new(tpl.SafeMap[string, string]),
}

func (p *SettingImpl) WithTx(tx *gorm.DB) *SettingImpl {
	return &SettingImpl{
		orm:   db.NewORM[db.Setting](tx),
		cache: p.cache,
	}
}

func (s *SettingImpl) BuildCache() error {
	rec, err := s.orm.Find()
	if err != nil {
		return err
	}
	for _, v := range rec {
		s.cache.Set(v.Name, v.Value)
	}
	return nil
}

// GetAll return all settings.
//
// Copies are returned, modification will not be saved.
func (s *SettingImpl) GetAll() map[string]string {
	return s.cache.Map()
}

// Get get name setting.
func (s *SettingImpl) Get(name string) (string, bool) {
	return s.cache.Get(name)
}

// Set set setting name with value.
func (s *SettingImpl) Set(name string, value string) error {
	v, ok := s.cache.Get(name)
	if !ok {
		if err := s.orm.Create(&db.Setting{
			Name:  name,
			Value: value,
		}); err != nil {
			return err
		}
		s.cache.Set(name, value)
		return nil
	} else {
		if v != value {
			if err := s.orm.Where("name = ?", name).Update("value", value); err != nil {
				return err
			}
			s.cache.Set(name, value)
		}
	}
	return nil
}

// Delete delete name setting.
func (s *SettingImpl) Delete(name string) (bool, error) {
	if s.cache.Has(name) {
		row, err := s.orm.Where("name = ?", name).Delete()
		if err != nil {
			return false, err
		}
		s.cache.Delete(name)
		return row == 1, nil
	}
	return false, nil
}

// DeleteAll delete all settings.
func (s *SettingImpl) DeleteAll() (int64, error) {
	row, err := s.orm.DeleteAll()
	if err != nil {
		return 0, err
	}
	s.cache.Clear()
	return row, nil
}

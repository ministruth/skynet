package handler

import (
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils/tpl"

	"gorm.io/gorm"
)

type SettingImpl struct {
	orm   *sn.ORM[sn.Setting]
	cache *tpl.SafeMap[string, string]
}

func NewSettingHandler() sn.SettingHandler {
	return &SettingImpl{
		orm:   sn.NewORM[sn.Setting](sn.Skynet.DB),
		cache: new(tpl.SafeMap[string, string]),
	}
}

func (impl *SettingImpl) WithTx(tx *gorm.DB) sn.SettingHandler {
	return &SettingImpl{
		orm:   sn.NewORM[sn.Setting](tx),
		cache: impl.cache,
	}
}

func (impl *SettingImpl) BuildCache() error {
	rec, err := impl.orm.Find()
	if err != nil {
		return err
	}
	for _, v := range rec {
		impl.cache.Set(v.Name, v.Value)
	}
	return nil
}

func (impl *SettingImpl) GetAll() map[string]string {
	return impl.cache.Map()
}

func (impl *SettingImpl) Get(name string) (string, bool) {
	return impl.cache.Get(name)
}

func (impl *SettingImpl) Set(name string, value string) error {
	v, ok := impl.cache.Get(name)
	if !ok {
		if err := impl.orm.Create(&sn.Setting{
			Name:  name,
			Value: value,
		}); err != nil {
			return err
		}
		impl.cache.Set(name, value)
		return nil
	} else {
		if v != value {
			if err := impl.orm.Where("name = ?", name).Update("value", value); err != nil {
				return err
			}
			impl.cache.Set(name, value)
		}
	}
	return nil
}

func (impl *SettingImpl) Delete(name string) (bool, error) {
	if impl.cache.Has(name) {
		row, err := impl.orm.Where("name = ?", name).Delete()
		if err != nil {
			return false, err
		}
		impl.cache.Delete(name)
		return row == 1, nil
	}
	return false, nil
}

func (impl *SettingImpl) DeleteAll() (int64, error) {
	row, err := impl.orm.DeleteAll()
	if err != nil {
		return 0, err
	}
	impl.cache.Clear()
	return row, nil
}

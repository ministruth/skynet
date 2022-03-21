package impl

import (
	"context"
	"errors"
	"log"
	"skynet/db"
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

func HashPass(pass string) string {
	return utils.MD5(viper.GetString("database.salt_prefix") + pass + viper.GetString("database.salt_suffix"))
}

func ConnectDB() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(viper.GetInt("database.timeout")))
	defer cancel()

	switch viper.GetString("database.type") {
	case "sqlite":
		sn.Skynet.DB = db.NewDB(ctx, &db.DBConfig{
			Type: db.DBType_Sqlite,
			Path: viper.GetString("database.path"),
		})
	default:
		log.Fatalf("Database type %s not supported", viper.GetString("database.type"))
	}
}

func ConnectRedis() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(viper.GetInt("redis.timeout")))
	defer cancel()

	sn.Skynet.Redis = db.NewRedis(ctx, &db.RedisConfig{
		Address:  viper.GetString("redis.address"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})
}

func ConnectSession() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(viper.GetInt("session.timeout")))
	defer cancel()

	sn.Skynet.Session = db.NewSession(ctx, &db.SessionConfig{
		RedisClient: sn.Skynet.GetRedis(),
		Prefix:      viper.GetString("session.prefix"),
	})
}

// ParseCondition parse sn.SNCondition to gorm.DB object.
func ParseCondition(cond *sn.SNCondition, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = sn.Skynet.GetDB()
	}
	if cond == nil {
		return tx
	}

	for _, v := range cond.Order {
		tx = tx.Order(v)
	}
	if len(cond.Distinct) != 0 {
		tx = tx.Distinct(cond.Distinct...)
	}
	if cond.Limit != nil {
		tx = tx.Limit(cond.Limit.(int))
	}
	if cond.Offset != nil {
		tx = tx.Offset(cond.Offset.(int))
	}
	tx = tx.Where(cond.Query, cond.Args...)
	return tx
}

// TODO: change when gorm support generic

type ORMImpl[T sn.DBStruct] struct {
	tx *gorm.DB
}

type ORM[T sn.DBStruct] struct {
	Impl *ORMImpl[T]
}

func NewORMImpl[T sn.DBStruct](tx *gorm.DB) *ORMImpl[T] {
	if tx == nil {
		tx = sn.Skynet.GetDB()
	}
	return &ORMImpl[T]{
		tx: tx,
	}
}

func NewORM[T sn.DBStruct](tx *gorm.DB) *ORM[T] {
	return &ORM[T]{
		Impl: NewORMImpl[T](tx),
	}
}

func (o *ORMImpl[T]) Where(query any, args ...any) *ORMImpl[T] {
	return NewORMImpl[T](o.tx.Where(query, args...))
}

func (o *ORMImpl[T]) Joins(query string, args ...any) *ORMImpl[T] {
	return NewORMImpl[T](o.tx.Joins(query, args...))
}

func (o *ORMImpl[T]) ID(id uuid.UUID) *ORMImpl[T] {
	return NewORMImpl[T](o.tx.Where("id = ?", id))
}

func (o *ORMImpl[T]) Save(value *T) error {
	return tracerr.Wrap(o.tx.Save(value).Error)
}

func (o *ORMImpl[T]) Updates(column []string, value *T) error {
	if column == nil {
		return tracerr.Wrap(o.tx.Model(new(T)).Updates(value).Error)
	} else {
		return tracerr.Wrap(o.tx.Model(new(T)).Select(column).Updates(value).Error)
	}
}

func (o *ORMImpl[T]) Update(column string, value any) error {
	return tracerr.Wrap(o.tx.Model(new(T)).Update(column, value).Error)
}

func (o *ORMImpl[T]) Delete(conds ...any) (int64, error) {
	ret := o.tx.Delete(new(T), conds...)
	return ret.RowsAffected, tracerr.Wrap(ret.Error)
}

func (o *ORMImpl[T]) DeleteAll() (int64, error) {
	ret := o.tx.Where("1 = 1").Delete(new(T))
	return ret.RowsAffected, tracerr.Wrap(ret.Error)
}

func (o *ORMImpl[T]) Find(conds ...any) (ret []*T, err error) {
	err = tracerr.Wrap(o.tx.Find(&ret, conds...).Error)
	return
}

func (o *ORMImpl[T]) Creates(value []*T) error {
	return tracerr.Wrap(o.tx.Create(&value).Error)
}

func (o *ORMImpl[T]) Create(value *T) error {
	return tracerr.Wrap(o.tx.Create(value).Error)
}

func (o *ORMImpl[T]) Cond(cond *sn.SNCondition) *ORMImpl[T] {
	return NewORMImpl[T](ParseCondition(cond, o.tx))
}

func (o *ORMImpl[T]) Count(cond *sn.SNCondition) (int64, error) {
	var count int64
	if err := tracerr.Wrap(ParseCondition(cond, o.tx.Model(new(T))).
		Count(&count).Error); err != nil {
		return 0, err
	}
	return count, nil
}

func (o *ORMImpl[T]) Take(conds ...any) (ret *T, err error) {
	ret = new(T)
	err = tracerr.Wrap(o.tx.Take(ret, conds...).Error)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ret = nil
			err = nil
		} else {
			ret = nil
		}
	}
	return
}

func (o *ORM[T]) Delete(id uuid.UUID) (bool, error) {
	row, err := o.Impl.Delete(id)
	return row == 1, err
}

func (o *ORM[T]) DeleteAll() (int64, error) {
	return o.Impl.DeleteAll()
}

func (o *ORM[T]) Count(cond *sn.SNCondition) (int64, error) {
	return o.Impl.Count(cond)
}

func (o *ORM[T]) Get(id uuid.UUID) (*T, error) {
	return o.Impl.Take(id)
}

func (o *ORM[T]) GetAll(cond *sn.SNCondition) (ret []*T, err error) {
	return o.Impl.Cond(cond).Find()
}

func (o *ORM[T]) TX() *gorm.DB {
	return o.Impl.tx
}

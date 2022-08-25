package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

// Condition limit condition search, however, sqli is not protected.
//
// Unprotected fields: Order, Distinct, Where(when not use ? as argument form)
//
// Warning: Caller should check user input on their OWN!
type Condition struct {
	Order    []any
	Distinct []any
	Limit    any
	Offset   any
	Query    string
	Args     []any
}

func (c *Condition) merge(in *Condition) {
	c.Order = append(c.Order, in.Order...)
	c.Distinct = append(c.Distinct, in.Distinct...)
	if c.Limit == nil {
		c.Limit = in.Limit
	}
	if c.Offset == nil {
		c.Offset = in.Offset
	}
}

func (c *Condition) MergeAnd(in *Condition) {
	if in == nil {
		return
	}
	c.merge(in)
	c.parseQuery(false, in.Query, in.Args...)
}

func (c *Condition) MergeOr(in *Condition) {
	if in == nil {
		return
	}
	c.merge(in)
	c.parseQuery(true, in.Query, in.Args...)
}

func (c *Condition) parseQuery(or bool, query string, args ...any) {
	if query == "" {
		return
	}
	if c.Query == "" {
		c.Query = query
		c.Args = args
	} else {
		if or {
			c.Query = fmt.Sprintf("(%v) OR (%v)", c.Query, query)
		} else {
			c.Query = fmt.Sprintf("(%v) AND (%v)", c.Query, query)
		}
		c.Args = append(c.Args, args...)
	}
}

func (c *Condition) AndLike(query string, arg string) {
	if arg != "" {
		c.And(query)
		count := strings.Count(query, "?")
		for i := 0; i < count; i++ {
			c.Args = append(c.Args, "%"+arg+"%")
		}
	}
}

func (c *Condition) OrLike(query string, arg string) {
	if arg != "" {
		c.Or(query)
		count := strings.Count(query, "?")
		for i := 0; i < count; i++ {
			c.Args = append(c.Args, "%"+arg+"%")
		}
	}
}

func (c *Condition) And(query string, args ...any) {
	c.parseQuery(false, query, args...)
}

func (c *Condition) Or(query string, args ...any) {
	c.parseQuery(true, query, args...)
}

// ParseCondition parse Condition to gorm.DB object.
func ParseCondition(cond *Condition, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = DB
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

type ORM[T DBStruct] struct {
	tx *gorm.DB
}

func NewORM[T DBStruct](tx *gorm.DB) *ORM[T] {
	if tx == nil {
		tx = DB
	}
	return &ORM[T]{
		tx: tx,
	}
}

func (o *ORM[T]) Where(query any, args ...any) *ORM[T] {
	return NewORM[T](o.tx.Where(query, args...))
}

func (o *ORM[T]) Joins(query string, args ...any) *ORM[T] {
	return NewORM[T](o.tx.Joins(query, args...))
}

func (o *ORM[T]) ID(id uuid.UUID) *ORM[T] {
	return NewORM[T](o.tx.Where("id = ?", id))
}

func (o *ORM[T]) Save(value *T) error {
	return tracerr.Wrap(o.tx.Save(value).Error)
}

func (o *ORM[T]) Updates(column []string, value *T) error {
	if column == nil {
		return tracerr.Wrap(o.tx.Model(new(T)).Updates(value).Error)
	} else {
		return tracerr.Wrap(o.tx.Model(new(T)).Select(column).Updates(value).Error)
	}
}

func (o *ORM[T]) Update(column string, value any) error {
	return tracerr.Wrap(o.tx.Model(new(T)).Update(column, value).Error)
}

func (o *ORM[T]) Delete(conds ...any) (int64, error) {
	ret := o.tx.Delete(new(T), conds...)
	return ret.RowsAffected, tracerr.Wrap(ret.Error)
}

func (o *ORM[T]) DeleteID(id uuid.UUID) (bool, error) {
	ret := o.tx.Delete(new(T), id)
	return ret.RowsAffected == 1, tracerr.Wrap(ret.Error)
}

func (o *ORM[T]) DeleteAll() (int64, error) {
	ret := o.tx.Where("1 = 1").Delete(new(T))
	return ret.RowsAffected, tracerr.Wrap(ret.Error)
}

func (o *ORM[T]) Find(conds ...any) (ret []*T, err error) {
	err = tracerr.Wrap(o.tx.Find(&ret, conds...).Error)
	return
}

func (o *ORM[T]) Creates(value []*T) error {
	return tracerr.Wrap(o.tx.Create(&value).Error)
}

func (o *ORM[T]) Create(value *T) error {
	return tracerr.Wrap(o.tx.Create(value).Error)
}

func (o *ORM[T]) Cond(cond *Condition) *ORM[T] {
	return NewORM[T](ParseCondition(cond, o.tx))
}

func (o *ORM[T]) Count(cond *Condition) (int64, error) {
	var count int64
	if err := tracerr.Wrap(ParseCondition(cond, o.tx.Model(new(T))).
		Count(&count).Error); err != nil {
		return 0, err
	}
	return count, nil
}

func (o *ORM[T]) Take(conds ...any) (ret *T, err error) {
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

func (o *ORM[T]) TX() *gorm.DB {
	return o.tx
}

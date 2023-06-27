package sn

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

// Condition limits condition search. However, sqli is not protected.
//
// Unprotected fields: Order, Distinct, Where(when not use ? as argument form)
//
// Warning: Caller should check user input on their OWN!
type Condition struct {
	Order    []any
	Distinct []any
	Limit    int
	Offset   int

	Query string
	Args  []any
}

func (c *Condition) merge(in *Condition) {
	c.Order = append(c.Order, in.Order...)
	c.Distinct = append(c.Distinct, in.Distinct...)
	c.Limit = utils.Max(c.Limit, in.Limit)
	c.Offset = utils.Max(c.Offset, in.Offset)
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

func (c *Condition) filter(str string) string {
	str = strings.ReplaceAll(str, "\\", "\\\\") // first replace \
	str = strings.ReplaceAll(str, "%", "\\%")
	str = strings.ReplaceAll(str, "_", "\\_")
	return str
}

func (c *Condition) AndLike(query string, arg string) {
	if arg != "" {
		query = strings.ReplaceAll(query, "?", "? escape '\\'")
		c.And(query)
		count := strings.Count(query, "? escape")
		arg = c.filter(arg)
		for i := 0; i < count; i++ {
			c.Args = append(c.Args, "%"+arg+"%")
		}
	}
}

func (c *Condition) OrLike(query string, arg string) {
	if arg != "" {
		query = strings.ReplaceAll(query, "?", "? escape '\\'")
		c.Or(query)
		count := strings.Count(query, "? escape")
		arg = c.filter(arg)
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
	if cond == nil {
		return tx
	}

	for _, v := range cond.Order {
		tx = tx.Order(v)
	}
	if len(cond.Distinct) != 0 {
		tx = tx.Distinct(cond.Distinct...)
	}
	if cond.Limit != 0 {
		tx = tx.Limit(cond.Limit)
	}
	if cond.Offset != 0 {
		tx = tx.Offset(cond.Offset)
	}
	if cond.Query != "" {
		tx = tx.Where(cond.Query, cond.Args...)
	}
	return tx
}

type ORM[T DBStruct] struct {
	tx *gorm.DB
}

func NewORM[T DBStruct](tx *gorm.DB) *ORM[T] {
	if tx == nil {
		log.New().Panic("tx should not be nil")
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
	var tmpCond *Condition
	if cond != nil {
		tmpCond = &Condition{
			Order:    cond.Order,
			Distinct: cond.Distinct,
			Limit:    0,
			Offset:   0,
			Query:    cond.Query,
			Args:     cond.Args,
		}
	}
	if err := tracerr.Wrap(ParseCondition(tmpCond, o.tx.Model(new(T))).
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

func (o *ORM[T]) Transaction(f func(*gorm.DB) error) error {
	return o.tx.Transaction(f)
}

func (o *ORM[T]) Tx() *gorm.DB {
	return o.tx
}

func (o *ORM[T]) WithTx(tx *gorm.DB) *ORM[T] {
	return NewORM[T](tx)
}

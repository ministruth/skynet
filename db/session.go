package db

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"

	"github.com/google/uuid"
	"github.com/rbcervilla/redisstore/v9"
	"github.com/ztrue/tracerr"
)

func init() {
	gob.Register(uuid.UUID{})
}

type SessionImpl struct {
	store   *redisstore.RedisStore
	timeout time.Duration
	prefix  string
}

func (s *SessionImpl) GetStore() *redisstore.RedisStore {
	return s.store
}

func (s *SessionImpl) getData(key []string) ([]*sn.SessionData, error) {
	pipe := sn.Skynet.Redis.Pipeline()
	piperes := make(map[string]*redis.StringCmd)
	for _, v := range key {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		piperes[v] = pipe.Get(ctx, v)
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	for _, v := range piperes {
		if v.Err() != nil {
			return nil, tracerr.Wrap(v.Err())
		}
	}

	// since redis store gob serialized data, we need to decode first
	var ret []*sn.SessionData
	for _, v := range piperes {
		b, err := v.Bytes()
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		tmp, err := LoadSessionBytes(b)
		if err != nil {
			return nil, err
		}
		ret = append(ret, tmp)
	}
	return ret, nil
}

func (s *SessionImpl) getKeys(uid []uuid.UUID) ([]string, error) {
	var res []string
	var err error
	if uid == nil {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		res, err = sn.Skynet.Redis.Keys(ctx, fmt.Sprintf("%v*", s.prefix)).Result()
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
	} else {
		for _, v := range uid {
			ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
			defer cancel()
			ids, err := sn.Skynet.Redis.Keys(ctx, fmt.Sprintf("%v%v_*", s.prefix, strings.ToUpper(v.String()))).Result()
			if err != nil {
				return nil, tracerr.Wrap(err)
			}
			res = append(ids, ids...)
		}
	}
	return res, nil
}

func (s *SessionImpl) Find(uid []uuid.UUID) (map[string][]*sn.SessionData, error) {
	res, err := s.getKeys(uid)
	if err != nil {
		return nil, err
	}
	ret := make(map[string][]*sn.SessionData)
	if len(res) == 0 {
		return ret, nil
	}
	data, err := s.getData(res)
	if err != nil {
		return nil, err
	}
	for _, v := range data {
		ret[v.ID.String()] = append(ret[v.ID.String()], v)
	}
	return ret, nil
}

func (s *SessionImpl) Delete(uid []uuid.UUID) error {
	res, err := s.getKeys(uid)
	if err != nil {
		return err
	}

	pipe := sn.Skynet.Redis.Pipeline()
	piperes := make(map[string]*redis.IntCmd)
	for _, v := range res {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		piperes[v] = pipe.Del(ctx, v)
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	_, err = pipe.Exec(ctx)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, v := range piperes {
		if v.Err() != nil {
			return tracerr.Wrap(v.Err())
		}
	}
	return nil
}

// NewSession connect session with config.
func NewSession(redis *redis.Client, prefix string, timeout time.Duration) (sn.Session, error) {
	log.New().WithFields(log.F{
		"prefix":  prefix,
		"timeout": timeout,
	}).Debug("Initializing session")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	store, err := redisstore.NewRedisStore(ctx, redis)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	store.KeyPrefix(prefix)
	log.New().Debug("Session connected")
	return &SessionImpl{
		store:   store,
		timeout: timeout,
		prefix:  prefix,
	}, nil
}

func LoadSession(session *sessions.Session) (*sn.SessionData, error) {
	var ret sn.SessionData
	var ok bool
	ret.ID, ok = session.Values["id"].(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%w: id not found", sn.ErrSessionInvalid)
	}
	ret.Time, ok = session.Values["time"].(int64)
	if !ok {
		return nil, fmt.Errorf("%w: time not found", sn.ErrSessionInvalid)
	}
	return &ret, nil
}

func LoadSessionBytes(b []byte) (*sn.SessionData, error) {
	tmp := new(sessions.Session)
	dec := gob.NewDecoder(bytes.NewReader(b))
	if err := tracerr.Wrap(dec.Decode(&tmp.Values)); err != nil {
		return nil, err
	}
	return LoadSession(tmp)
}

// GetCTXSession gets session object from gin context.
func GetCTXSession(c *gin.Context) (*sessions.Session, error) {
	ret, err := sn.Skynet.Session.GetStore().Get(c.Request, viper.GetString("session.cookie"))
	return ret, tracerr.Wrap(err)
}

// SaveCTXSession saves session object to gin context.
func SaveCTXSession(c *gin.Context) error {
	return tracerr.Wrap(sessions.Save(c.Request, c.Writer))
}

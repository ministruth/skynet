package main

import (
	"context"
	"errors"
	plugins "skynet/plugin"
	monitor "skynet/plugin/monitor/shared"
	"skynet/plugin/task/shared"
	"skynet/sn"
	"skynet/sn/utils"

	log "github.com/sirupsen/logrus"
)

func NewShared() shared.PluginShared {
	return &pluginShared{}
}

type pluginShared struct{}

var (
	ErrLoadMonitorPlugin    = errors.New("monitor plugin not available")
	ErrTaskNotSupportCancel = errors.New("task not support cancel")
)

func (s *pluginShared) GetInstance() *plugins.PluginInstance {
	return Instance
}

func (s *pluginShared) New(name string, detail string, cancel func() error) (int, error) {
	rec := shared.PluginTask{
		Name:   name,
		Detail: detail,
	}
	err := utils.GetDB().Create(&rec).Error
	if err != nil {
		return 0, err
	}
	if cancel != nil {
		taskCancel.Set(int(rec.ID), cancel)
	}
	return int(rec.ID), nil
}

func (s *pluginShared) Cancel(id int, msg string) error {
	if c, ok := taskCancel.Get(id); ok {
		if msg != "" {
			s.AppendOutputNewLine(id, msg)
		}
		return c()
	}
	return ErrTaskNotSupportCancel
}

func (s *pluginShared) CancelByUser(id int, msg string) error {
	if c, ok := taskCancel.Get(id); ok {
		s.UpdateStatus(id, shared.TaskStop)
		if msg != "" {
			s.AppendOutputNewLine(id, msg)
		}
		return c()
	}
	return ErrTaskNotSupportCancel
}

func (s *pluginShared) Get(id int) (*shared.PluginTask, error) {
	var ret shared.PluginTask
	err := utils.GetDB().First(&ret, id).Error
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (s *pluginShared) GetAll(order []interface{}, limit interface{}, offset interface{}, where interface{}, args ...interface{}) ([]*shared.PluginTask, error) {
	var ret []*shared.PluginTask
	db := utils.GetDB()
	for _, v := range order {
		db = db.Order(v)
	}
	if limit != nil {
		db = db.Limit(limit.(int))
	}
	if offset != nil {
		db = db.Offset(offset.(int))
	}
	if where != nil {
		db = db.Where(where, args...)
	}
	err := db.Find(&ret).Error
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *pluginShared) Count() (int64, error) {
	var count int64
	err := utils.GetDB().Model(&shared.PluginTask{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *pluginShared) AppendOutput(id int, out string) error {
	rec, err := s.Get(id)
	if err != nil {
		return err
	}
	rec.Output += out
	err = utils.GetDB().Save(rec).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *pluginShared) AppendOutputNewLine(id int, out string) error {
	rec, err := s.Get(id)
	if err != nil {
		return err
	}
	if rec.Output != "" {
		rec.Output += "\n"
	}
	rec.Output += out
	err = utils.GetDB().Save(rec).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *pluginShared) UpdateOutput(id int, out string) error {
	rec, err := s.Get(id)
	if err != nil {
		return err
	}
	rec.Output = out
	err = utils.GetDB().Save(rec).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *pluginShared) UpdateStatus(id int, status shared.TaskStatus) error {
	rec, err := s.Get(id)
	if err != nil {
		return err
	}
	rec.Status = status
	err = utils.GetDB().Save(rec).Error
	if err != nil {
		return err
	}
	if status == shared.TaskFail || status == shared.TaskStop || status == shared.TaskSuccess {
		taskCancel.Delete(id)
	}
	return nil
}

func (s *pluginShared) AddPercent(id int, percent int) error {
	rec, err := s.Get(id)
	if err != nil {
		return err
	}
	rec.Percent += int32(percent)
	err = utils.GetDB().Save(rec).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *pluginShared) UpdatePercent(id int, percent int) error {
	rec, err := s.Get(id)
	if err != nil {
		return err
	}
	rec.Percent = int32(percent)
	err = utils.GetDB().Save(rec).Error
	if err != nil {
		return err
	}
	return nil
}

func (s *pluginShared) NewCommand(agentID int, cmd string, name string, detail string) (chan bool, error) {
	m, exist := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
	if !exist {
		return nil, ErrLoadMonitorPlugin
	}
	uid, resChan, err := m.RunCMDAsync(agentID, cmd)
	if err != nil {
		return nil, err
	}

	tid, err := s.New(name, detail, func() error {
		return m.KillCMD(agentID, uid)
	})
	if err != nil {
		return nil, err
	}
	s.UpdateStatus(tid, shared.TaskRunning)
	fini := make(chan bool)

	go func() {
		for res := range resChan {
			s.AppendOutput(tid, res)
		}
		res, err := m.GetCMDRes(agentID, uid)
		if err != nil || !res.End {
			s.UpdateStatus(tid, shared.TaskFail)
			if err != nil {
				s.AppendOutputNewLine(tid, err.Error())
			} else {
				s.AppendOutputNewLine(tid, "Unknown error")
			}
			fini <- false
			close(fini)
			return
		}
		if res.Complete {
			if res.Code != 0 {
				s.UpdateStatus(tid, shared.TaskFail)
			} else {
				s.UpdateStatus(tid, shared.TaskSuccess)
			}
		}
		fini <- res.Complete
		close(fini)
	}()
	return fini, nil
}

func (s *pluginShared) NewCustom(agentID int, name string, detail string, c func() error, f func(ctx context.Context, agentID int, taskID int) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	tid, err := s.New(name, detail, func() error {
		if c != nil {
			c()
		}
		cancel()
		return nil
	})
	if err != nil {
		cancel()
		return err
	}
	go func() {
		err := f(ctx, agentID, tid)
		if errors.Is(err, context.Canceled) {
			s.CancelByUser(tid, "Task killed by user")
		} else if err != nil {
			log.WithFields(defaultField).Error(err)
			s.Cancel(tid, "Task fail: "+err.Error())
			s.UpdateStatus(tid, shared.TaskFail)
		}
		cancel()
	}()
	return nil
}

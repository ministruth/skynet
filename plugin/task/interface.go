package main

import (
	"errors"
	monitor "skynet/plugin/monitor/shared"
	"skynet/plugin/task/shared"
	"skynet/sn"
	"skynet/sn/utils"
)

func NewShared() shared.PluginShared {
	return &pluginShared{}
}

type pluginShared struct{}

func (s *pluginShared) New(name string, detail string, cancel func()) (int, error) {
	rec := shared.PluginTasks{
		Name:   name,
		Detail: detail,
	}
	err := utils.GetDB().Create(&rec).Error
	if err != nil {
		return 0, err
	}
	taskCancel[int(rec.ID)] = cancel
	return int(rec.ID), nil
}

func (s *pluginShared) Cancel(id int) {
	if c, exist := taskCancel[id]; exist {
		c()
	}
}

func (s *pluginShared) Get(id int) (*shared.PluginTasks, error) {
	var ret shared.PluginTasks
	err := utils.GetDB().First(&ret, id).Error
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (s *pluginShared) GetAll(order []interface{}, limit interface{}, offset interface{}, where interface{}, args ...interface{}) ([]*shared.PluginTasks, error) {
	var ret []*shared.PluginTasks
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
	err := utils.GetDB().Model(&shared.PluginTasks{}).Count(&count).Error
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
		delete(taskCancel, id)
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
		return nil, errors.New("monitor plugin not available")
	}
	uid, resChan, err := m.RunCMD(agentID, cmd)
	if err != nil {
		return nil, err
	}

	tid, err := s.New(name, detail, func() {
		m.KillCMD(agentID, uid)
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
		if !res.Complete {
			s.UpdateStatus(tid, shared.TaskStop)
			s.AppendOutputNewLine(tid, "Task force killed by user")
		} else {
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

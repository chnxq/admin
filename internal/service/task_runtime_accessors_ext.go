package service

import (
	taskruntime "admin/internal/task/runtime"
	"github.com/chnxq/xkitmod/log"
)

func (s *TaskService) TaskScheduler() *taskruntime.Scheduler {
	if s == nil {
		return nil
	}
	return s.scheduler
}

func (s *TaskService) TaskLogger() *log.Helper {
	if s == nil {
		return nil
	}
	return s.log
}

func (s *TaskGroupService) TaskScheduler() *taskruntime.Scheduler {
	if s == nil {
		return nil
	}
	return s.scheduler
}

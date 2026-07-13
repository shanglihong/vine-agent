package scheduler

import (
	"context"
	"github.com/robfig/cron/v3"
	"vine-agent/cmd/scheduler/job"
	"vine-agent/utils"
)

type Scheduler struct {
	cron *cron.Cron
}

func New() *Scheduler {
	scheduler := &Scheduler{
		cron: cron.New(),
	}
	return scheduler
}

// Start 启动定时任务调度器
func (s *Scheduler) Start() *Scheduler {
	s.cron.Start()
	return s
}

// Stop 优雅关闭定时任务调度器
func (s *Scheduler) Stop() *Scheduler {
	s.cron.Stop()
	return s
}

func (s *Scheduler) RegisterJob(cron string, f func(ctx context.Context)) {
	_, err := s.cron.AddFunc(cron, func() {
		f(context.Background())
	})
	utils.Panic(err)
}

func (s *Scheduler) Register() *Scheduler {
	s.RegisterJob("*/1 * * * *", job.GetEvolutionJob().Run)
	return s
}

package scheduler

import (
	"context"
	"github.com/robfig/cron/v3"
	"log"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/scheduler/job"
	"vine-agent/utils"
)

type Scheduler struct {
	cron   *cron.Cron
	domain *bootstrap.DomainContainer
	app    *bootstrap.AppContainer
}

func NewScheduler(domain *bootstrap.DomainContainer, app *bootstrap.AppContainer) *Scheduler {
	scheduler := &Scheduler{
		cron:   cron.New(),
		domain: domain,
		app:    app,
	}
	scheduler.registerJob()
	return scheduler
}

// Start 启动定时任务调度器
func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("[CronScheduler] 定时任务调度器已成功启动")
}

// Stop 优雅关闭定时任务调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("[CronScheduler] 定时任务调度器已优雅停止")
}

func (s *Scheduler) Register(cron string, f func(ctx context.Context)) {
	_, err := s.cron.AddFunc(cron, func() {
		f(context.Background())
	})
	utils.Panic(err)
}

// 统一管理注册
func (s *Scheduler) registerJob() {
	s.Register("*/1 * * * *", job.NewEvolutionJob(s.domain, s.app).Run)
}

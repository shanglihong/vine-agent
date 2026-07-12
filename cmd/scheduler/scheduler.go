package scheduler

import (
	"context"
	"github.com/robfig/cron/v3"
	"log"
	"vine-agent/utils"
)

// CronScheduler 统一管理系统中所有的定时任务调度
type CronScheduler struct {
	cron *cron.Cron
}

// NewCronScheduler 构造一个新的 CronScheduler
func NewCronScheduler() *CronScheduler {
	return &CronScheduler{
		cron: cron.New(),
	}
}

func (s *CronScheduler) Register(cron string, doFunc func(ctx context.Context)) {
	_, err := s.cron.AddFunc(cron, func() {
		doFunc(context.Background())
	})
	utils.Panic(err)
}

// Start 启动定时任务调度器
func (s *CronScheduler) Start() {
	s.cron.Start()
	log.Println("[CronScheduler] 定时任务调度器已成功启动")
}

// Stop 优雅关闭定时任务调度器
func (s *CronScheduler) Stop() {
	s.cron.Stop()
	log.Println("[CronScheduler] 定时任务调度器已优雅停止")
}

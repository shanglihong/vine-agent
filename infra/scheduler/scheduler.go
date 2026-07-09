package scheduler

import (
	"context"
	"log"
	"time"

	"vine-agent/app/memory"
	"vine-agent/domain/memory/session"

	"github.com/robfig/cron/v3"
)

// CronScheduler 统一管理系统中所有的定时任务调度
type CronScheduler struct {
	cron   *cron.Cron
	logger *log.Logger
}

// NewCronScheduler 构造一个新的 CronScheduler
func NewCronScheduler(logger *log.Logger) *CronScheduler {
	return &CronScheduler{
		cron:   cron.New(cron.WithLogger(cron.VerbosePrintfLogger(logger))),
		logger: logger,
	}
}

// Start 启动定时任务调度器
func (s *CronScheduler) Start() {
	s.cron.Start()
	s.logger.Println("[CronScheduler] 定时任务调度器已成功启动")
}

// Stop 优雅关闭定时任务调度器
func (s *CronScheduler) Stop() {
	s.cron.Stop()
	s.logger.Println("[CronScheduler] 定时任务调度器已优雅停止")
}

// RegisterEvolutionJob 注册记忆进化定时任务
func RegisterEvolutionJob(
	s *CronScheduler,
	spec string,
	sessionSvc session.SessionService,
	evolutionApp *memory.EvolutionAppService,
) error {
	_, err := s.cron.AddFunc(spec, func() {
		ctx := context.Background()
		// 由于是每分钟运行一次，这里取过去 2 分钟更新过的会话作为滑动时间窗
		since := time.Now().Add(-2 * time.Minute)

		sessions, err := sessionSvc.ListUpdatedSince(ctx, since)
		if err != nil {
			s.logger.Printf("[EvolutionJob] 获取近期更新的会话列表失败: %v", err)
			return
		}
		if len(sessions) == 0 {
			return
		}

		s.logger.Printf("[EvolutionJob] 捞取到 %d 个可能需要进化的会话，开始检查增量消息...", len(sessions))
		for _, sess := range sessions {
			if err := evolutionApp.TriggerEvolution(ctx, sess.ID); err != nil {
				s.logger.Printf("[EvolutionJob] 会话 %s 记忆演进失败: %v", sess.ID, err)
			}
		}
	})
	return err
}

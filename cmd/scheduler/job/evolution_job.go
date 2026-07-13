package job

import (
	"context"
	"log"
	"time"
	"vine-agent/cmd/bootstrap"
)

var (
	evolutionJob = EvolutionJob{}
)

type EvolutionJob struct{}

func GetEvolutionJob() *EvolutionJob {
	return &evolutionJob
}

func (job *EvolutionJob) Run(ctx context.Context) {
	// 由于是每分钟运行一次，这里取过去 2 分钟更新过的会话作为滑动时间窗
	since := time.Now().Add(-60 * time.Minute)

	sessions, err := bootstrap.GetDomainContainer().SessionService.ListUpdatedSince(ctx, since)
	if err != nil {
		log.Printf("[EvolutionJob] 获取近期更新的会话列表失败: %v", err)
		return
	}
	if len(sessions) == 0 {
		return
	}

	log.Printf("[EvolutionJob] 捞取到 %d 个可能需要进化的会话，提取 IDs 并触发批量进化...", len(sessions))
	sessionIDs := make([]string, len(sessions))
	for i, sess := range sessions {
		sessionIDs[i] = sess.ID
	}
	if err := bootstrap.GetAppContainer().EvolutionAppService.TriggerEvolution(ctx, sessionIDs); err != nil {
		log.Printf("[EvolutionJob] 批量触发记忆演进失败: %v", err)
	}
}

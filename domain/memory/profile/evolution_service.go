package profile

import (
	"context"

	"vine-agent/domain/message"
)

// evolutionService 长期记忆进化服务的具体私有实现，对外隐藏实现细节
type evolutionService struct {
	extractor Extractor
}

// NewEvolutionService 构造并返回一个 EvolutionService 接口实例
func NewEvolutionService(extractor Extractor) EvolutionService {
	return &evolutionService{
		extractor: extractor,
	}
}

// Evolve 编排提取器(Extractor)进行长期偏好与事实的提取，并触发聚合根 Profile 执行自我合并更新
func (s *evolutionService) Evolve(ctx context.Context, prof *Profile, messages []message.Message) error {
	if len(messages) == 0 {
		return nil
	}

	// 1. 调用提炼器，从对话消息中提取偏好和事实，并交由大模型完成合并
	newPrefs, newFacts, err := s.extractor.Extract(ctx, messages, prof.Preferences, prof.Facts)
	if err != nil {
		return err
	}

	// 2. 将大模型提炼合并后的最新列表更新到领域聚合根中
	prof.Update(newPrefs, newFacts)

	return nil
}

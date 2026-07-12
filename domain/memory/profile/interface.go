package profile

import (
	"context"

	"vine-agent/domain/message"
)

//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/profile_mock.go -package=mock

// ProfileRepository 定义了用户长期记忆画像持久化的多态契约接口
type ProfileRepository interface {
	// GetByUserID 获取指定用户的记忆画像。如果不存在，返回 (nil, nil)
	GetByUserID(ctx context.Context, userID string) (*Profile, error)

	// Save 保存用户的记忆画像
	Save(ctx context.Context, prof *Profile) error
}

// Extractor 定义了大模型或其它机制从中提取偏好与事实的通用接口契约
type Extractor interface {
	// Extract 解析给定的对话消息，结合已有的偏好和事实，提取并合并输出最新的偏好和事实列表
	Extract(ctx context.Context, messages []message.Message, existingPrefs []string, existingFacts []string) ([]string, []string, error)
}

// EvolutionService 领域服务，编排将新消息提取偏好/事实并合并进已有画像的过程
type EvolutionService interface {
	// Evolve 核心记忆进化逻辑。它将调用 Extractor 从增量消息中提炼偏好和事实，并合并至现有 Profile
	Evolve(ctx context.Context, prof *Profile, messages []message.Message) error
}

type ProfileService interface {
	// GetByUserID 获取指定用户的记忆画像。如果不存在，返回 (nil, nil)
	GetByUserID(ctx context.Context, userID string) (*Profile, error)
	Save(ctx context.Context, prof *Profile) error
}

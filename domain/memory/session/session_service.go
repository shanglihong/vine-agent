package session

import (
	"context"
	"fmt"
	"time"

	"vine-agent/domain/message"
	"vine-agent/utils"
)

const defaultTTL = 5 * time.Minute

// SessionService
type SessionService struct {
	persist Repository
	cache   *utils.TTLCache[*Session]
}

// SessionStoreOption 用于配置 SessionStore 服务的选项函数类型
type SessionStoreOption func(*SessionService)

// WithSessionTTL 配置缓存的生存时间 (Time To Live)。
// 如果传入的 ttl <= 0，则表示不启用过期机制（即缓存永久有效）。
func WithSessionTTL(ttl time.Duration) SessionStoreOption {
	return func(s *SessionService) {
		s.cache.SetTTL(ttl)
	}
}

// NewSessionStore 创建一个包含缓存机制 of SessionStore 领域服务实例，支持配置过期时间 (TTL)
func NewSessionStore(persist Repository, opts ...SessionStoreOption) *SessionService {
	s := &SessionService{
		persist: persist,
		cache:   utils.NewTTLCache[*Session](defaultTTL),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Save 保存 Session 到物理存储中，并使内存缓存中的对应记录失效以保持数据一致性
func (s *SessionService) Save(ctx context.Context, sess *Session) error {
	if sess == nil {
		return fmt.Errorf("session cannot be nil")
	}

	sess.UpdatedAt = time.Now()

	if err := s.persist.Save(ctx, sess); err != nil {
		return err
	}

	s.cache.Delete(sess.ID)
	return nil
}

// Get 首先尝试从并发安全内存缓存中读取 Session，如果未命中则穿透到底层物理持久化中读取并缓存
func (s *SessionService) Get(ctx context.Context, id string) (*Session, error) {
	if sess, ok := s.cache.Get(id); ok {
		return cloneSession(sess), nil
	}

	sess, err := s.persist.Get(ctx, id)
	if err != nil {
		return nil, err // 直接透传底层 Repository 返回 of 领域错误
	}

	s.cache.Set(id, sess)
	return cloneSession(sess), nil
}

// Delete 从物理持久化中删除 Session，并清理内存缓存中的记录
func (s *SessionService) Delete(ctx context.Context, id string) error {
	if err := s.persist.Delete(ctx, id); err != nil {
		return err // 直接透传底层 Repository 返回 of 领域错误
	}
	s.cache.Delete(id)
	return nil
}

// List 从物理持久化中拉取用户会话列表（不携带冗余的历史消息详情），此方法不走缓存
func (s *SessionService) List(ctx context.Context, userID string) ([]*Session, error) {
	return s.persist.List(ctx, userID)
}

// cloneSession 实现 Session 结构体的并发安全深拷贝
func cloneSession(s *Session) *Session {
	if s == nil {
		return nil
	}
	cloned := &Session{
		ID:        s.ID,
		UserID:    s.UserID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if s.Metadata != nil {
		cloned.Metadata = make(map[string]string, len(s.Metadata))
		for k, v := range s.Metadata {
			cloned.Metadata[k] = v
		}
	}
	if s.Messages != nil {
		cloned.Messages = make([]message.Message, len(s.Messages))
		copy(cloned.Messages, s.Messages)
	}
	return cloned
}

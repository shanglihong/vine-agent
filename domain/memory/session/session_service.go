package session

import (
	"context"
	"fmt"
	"time"

	"vine-agent/domain/message"
	"vine-agent/utils"
)

const defaultTTL = 5 * time.Minute

// sessionService
type sessionService struct {
	persist SessionRepository
	cache   *utils.TTLCache[*Session]
}

// NewSessionService 创建一个包含缓存机制的 sessionService 领域服务实例，返回 SessionService 接口
func NewSessionService(persist SessionRepository) SessionService {
	return &sessionService{
		persist: persist,
		cache:   utils.NewTTLCache[*Session](defaultTTL),
	}
}

// Save 保存 Session 到物理存储中，并使内存缓存中的对应记录失效以保持数据一致性
func (s *sessionService) Save(ctx context.Context, sess *Session) error {
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
func (s *sessionService) Get(ctx context.Context, id string) (*Session, error) {
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
func (s *sessionService) Delete(ctx context.Context, id string) error {
	if err := s.persist.Delete(ctx, id); err != nil {
		return err // 直接透传底层 Repository 返回 of 领域错误
	}
	s.cache.Delete(id)
	return nil
}

// List 从物理持久化中拉取用户会话列表（不携带冗余的历史消息详情），此方法不走缓存
func (s *sessionService) List(ctx context.Context, userID string) ([]*Session, error) {
	return s.persist.List(ctx, userID)
}

// Rename 重命名指定的会话并使缓存失效
func (s *sessionService) Rename(ctx context.Context, id string, name string) error {
	sess, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	sess.Name = name
	return s.Save(ctx, sess)
}

// ListUpdatedSince 从物理存储中列出在指定时间点之后更新过的所有会话，此方法不走缓存，列表不携带历史消息详情
func (s *sessionService) ListUpdatedSince(ctx context.Context, since time.Time) ([]*Session, error) {
	return s.persist.ListUpdatedSince(ctx, since)
}

// cloneSession 实现 Session 结构体的并发安全深拷贝
func cloneSession(s *Session) *Session {
	if s == nil {
		return nil
	}
	cloned := &Session{
		ID:        s.ID,
		UserID:    s.UserID,
		Name:      s.Name,
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

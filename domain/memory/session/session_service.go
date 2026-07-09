package session

import (
	"context"
	"fmt"
	"time"

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
		return sess.Clone(), nil
	}

	sess, err := s.persist.Get(ctx, id)
	if err != nil {
		return nil, err // 直接透传底层 Repository 返回 of 领域错误
	}

	s.cache.Set(id, sess)
	return sess.Clone(), nil
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



// GetBatch 批量获取 Session 实体。内部并发安全加载，优先读取缓存，未命中则穿透物理仓储批量获取
func (s *sessionService) GetBatch(ctx context.Context, ids []string) (map[string]*Session, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	res := make(map[string]*Session)
	var missingIDs []string

	// 1. 尝试从缓存获取
	for _, id := range ids {
		if sess, ok := s.cache.Get(id); ok {
			res[id] = sess.Clone()
		} else {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) == 0 {
		return res, nil
	}

	// 2. 批量从底层物理仓储获取未命中的
	dbSessions, err := s.persist.GetBatch(ctx, missingIDs)

	// 3. 把物理获取到的写入缓存，并加入最终结果
	for _, sess := range dbSessions {
		s.cache.Set(sess.ID, sess)
		res[sess.ID] = sess.Clone()
	}

	return res, err
}

package profile

import (
	"strings"
	"time"
)

// Profile 长期记忆画像聚合根，用于聚合用户的偏好列表与事实列表
type Profile struct {
	UserID      string    `json:"user_id"`
	Preferences []string  `json:"preferences"` // 偏好纯文本列表
	Facts       []string  `json:"facts"`       // 事实纯文本列表
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewProfile 初始化一个空的长期记忆画像
func NewProfile(userID string) *Profile {
	return &Profile{
		UserID:      userID,
		Preferences: make([]string, 0),
		Facts:       make([]string, 0),
		UpdatedAt:   time.Now(),
	}
}

// Update 接收大模型提炼合并后的最新偏好与事实列表，更新聚合根状态
func (p *Profile) Update(newPrefs []string, newFacts []string) {
	p.Preferences = make([]string, 0, len(newPrefs))
	for _, np := range newPrefs {
		npTrimmed := p.cleanItem(np)
		if npTrimmed != "" {
			p.Preferences = append(p.Preferences, npTrimmed)
		}
	}

	p.Facts = make([]string, 0, len(newFacts))
	for _, nf := range newFacts {
		nfTrimmed := p.cleanItem(nf)
		if nfTrimmed != "" {
			p.Facts = append(p.Facts, nfTrimmed)
		}
	}

	p.UpdatedAt = time.Now()
}

// cleanItem 清理字符串中的前导/尾随空格及不必要的 Markdown 列表符号（防御性）
func (p *Profile) cleanItem(item string) string {
	item = strings.TrimSpace(item)
	// 如果大模型提取出来的项还带有开头的 Markdown 符号，进行清除
	if strings.HasPrefix(item, "-") || strings.HasPrefix(item, "*") {
		item = strings.TrimLeft(item, "-* ")
	}
	return strings.TrimSpace(item)
}

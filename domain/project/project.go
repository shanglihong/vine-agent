package project

import (
	"fmt"
	"time"
)

// Project 代表项目聚合根实体，作为项目的顶级容器
type Project struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// NewProject 构造一个新的项目实例
func NewProject(id, userID, name, path, description string, metadata map[string]string) *Project {
	if id == "" {
		id = fmt.Sprintf("proj_%d", time.Now().UnixNano())
	}
	if metadata == nil {
		metadata = make(map[string]string)
	}
	now := time.Now()
	return &Project{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Path:        path,
		Description: description,
		Metadata:    metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Update 更新项目的核心属性和元数据
func (p *Project) Update(name, path, description string, metadata map[string]string) {
	p.Name = name
	p.Path = path
	p.Description = description
	if metadata != nil {
		p.Metadata = metadata
	}
	p.UpdatedAt = time.Now()
}

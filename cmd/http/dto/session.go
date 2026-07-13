package dto

import (
	"time"
	"vine-agent/domain/message"
)

type SessReq struct {
	SessionID string `json:"session_id" form:"session_id"`
	UserID    string `json:"user_id" form:"user_id"`
	ProjectID string `json:"project_id" form:"project_id"`
}

type CreateSessReq struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
}

type SessIdReq struct {
	SessionID string `uri:"id"`
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
}

type SessRenameReq struct {
	SessionID string `uri:"id"`
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

type SessResp struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	Name      string            `json:"name"`
	UpdatedAt time.Time         `json:"updated_at"`
	Status    string            `json:"status,omitempty"`
	ProjectID string            `json:"project_id,omitempty"`
	Messages  []message.Message `json:"messages,omitempty"`
}

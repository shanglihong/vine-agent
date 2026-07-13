package dto

import "time"

type SessReq struct {
	SessionID string `json:"session_id" form:"session_id" binding:"required"`
	UserID    string `json:"user_id" form:"user_id" binding:"required"`
	ProjectID string `json:"project_id" form:"project_id" binding:"required"`
}

type CreateSessReq struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id" binding:"required"`
	ProjectID string `json:"project_id" binding:"required"`
}

type SessIdReq struct {
	SessionID string `uri:"id" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	ProjectID string `json:"project_id" binding:"required"`
}

type SessRenameReq struct {
	SessionID string `uri:"id" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	ProjectID string `json:"project_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
}

type SessResp struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    string    `json:"status,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
}

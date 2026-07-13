package dto

type UserIdReq struct {
	UserID    string `uri:"id"`
	SessionID string `json:"session_id"`
}

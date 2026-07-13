package dto

type SessChatReq struct {
	SessionID string   `uri:"id"`
	UserID    string   `json:"user_id"`
	ProjectID string   `json:"project_id"`
	Message   string   `json:"message"`
	Model     string   `json:"model"`
	Tools     []string `json:"tools"`
}

type SessResumeChatReq struct {
	SessionID            string   `uri:"id"`
	UserID               string   `json:"user_id"`
	ConfirmedToolCallIDs []string `json:"confirmed_tool_call_ids"`
	Tools                []string `json:"tools"`
}

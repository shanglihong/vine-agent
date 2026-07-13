package dto

type CreatProjectReq struct {
	UserID      string            `json:"user_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

type ProjectUserIdReq struct {
	UserID string `json:"user_id" form:"user_id"`
}

type ProjectIdReq struct {
	ProjectId string `uri:"id"`
}

type ProjectUpdateReq struct {
	ProjectId   string `uri:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ProjectResp struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

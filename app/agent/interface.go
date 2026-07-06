package agent

import (
	"vine-agent/domain/chat"
)

//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/agent_mock.go -package=mock

// Service 智能体应用服务契约接口
type Service interface {
	chat.ChatModel
}

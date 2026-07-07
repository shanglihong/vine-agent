//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/user_mock.go -package=mock
package user

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

// UserRepository 定义了用户仓储多态契约接口
type UserRepository interface {
	Get(ctx context.Context, id string) (*User, error)
}

// UserService 定义了用户领域服务契约接口
type UserService interface {
	GetUser(ctx context.Context, id string) (*User, error)
}

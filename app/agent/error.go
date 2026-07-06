package agent

import (
	"fmt"
)

// TimeoutError 表示对话超时或被取消的错误
type TimeoutError struct {
	Err error
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("chat timeout or canceled: %v", e.Err)
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

// ToolExecutionError 表示工具执行出错的错误
type ToolExecutionError struct {
	ToolName string
	Err      error
}

func (e *ToolExecutionError) Error() string {
	return fmt.Sprintf("execute tool %q failed: %v", e.ToolName, e.Err)
}

func (e *ToolExecutionError) Unwrap() error {
	return e.Err
}

// ToolNotFoundError 表示工具未在配置中找到的错误
type ToolNotFoundError struct {
	ToolName string
}

func (e *ToolNotFoundError) Error() string {
	return fmt.Sprintf("tool %s not found in options", e.ToolName)
}

// ToolInvalidArgumentError 表示工具入参校验失败的错误
type ToolInvalidArgumentError struct {
	ToolName string
	Err      error
}

func (e *ToolInvalidArgumentError) Error() string {
	return fmt.Sprintf("invalid arguments for tool %q: %v", e.ToolName, e.Err)
}

func (e *ToolInvalidArgumentError) Unwrap() error {
	return e.Err
}

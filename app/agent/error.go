package agent

import (
	"errors"
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
	return fmt.Sprintf("tool %s execution failed: %v", e.ToolName, e.Err)
}

func (e *ToolExecutionError) Unwrap() error {
	return e.Err
}

// IsTimeoutError 判断错误是否为 TimeoutError
func IsTimeoutError(err error) bool {
	var target *TimeoutError
	return errors.As(err, &target)
}

// IsToolExecutionError 判断错误是否为 ToolExecutionError
func IsToolExecutionError(err error) bool {
	var target *ToolExecutionError
	return errors.As(err, &target)
}

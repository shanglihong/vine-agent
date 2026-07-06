package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	appagent "vine-agent/app/agent"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
)

type dummyTool struct {
	info    tool.Definition
	execute func(ctx context.Context, args string) (string, error)
}

func (t *dummyTool) Info() tool.Definition {
	return t.info
}

func (t *dummyTool) Execute(ctx context.Context, args string) (string, error) {
	if t.execute != nil {
		return t.execute(ctx, args)
	}
	return "", nil
}

func TestExecuteToolCall_Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("ToolNotFoundError", func(t *testing.T) {
		call := message.ToolCall{
			ID: "call_1",
			Function: message.FunctionCall{
				Name:      "test_tool",
				Arguments: `{}`,
			},
		}
		_, err := appagent.ExecuteToolCall(ctx, call, nil)
		assert.Error(t, err)
		
		var notFoundErr *appagent.ToolNotFoundError
		assert.True(t, errors.As(err, &notFoundErr))
		assert.Equal(t, "test_tool", notFoundErr.ToolName)
		assert.Contains(t, err.Error(), "tool test_tool not found in options")
	})

	t.Run("ToolInvalidArgumentError", func(t *testing.T) {
		// 定义需要参数为 string 的 schema，但传入非法的 JSON
		paramsSchema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type": "string",
				},
			},
			"required": []any{"name"},
		}
		tl := &dummyTool{
			info: tool.Definition{
				Name:       "dummy",
				Parameters: paramsSchema,
			},
		}

		call := message.ToolCall{
			ID: "call_2",
			Function: message.FunctionCall{
				Name:      "dummy",
				Arguments: `{"name": 123}`, // 类型错误，应为 string
			},
		}

		_, err := appagent.ExecuteToolCall(ctx, call, tl)
		assert.Error(t, err)

		var invalidArgErr *appagent.ToolInvalidArgumentError
		assert.True(t, errors.As(err, &invalidArgErr))
		assert.Equal(t, "dummy", invalidArgErr.ToolName)
		assert.Contains(t, err.Error(), "invalid arguments for tool")
	})

	t.Run("ToolExecutionError", func(t *testing.T) {
		expectedErr := errors.New("database connect fail")
		tl := &dummyTool{
			info: tool.Definition{
				Name: "dummy",
			},
			execute: func(ctx context.Context, args string) (string, error) {
				return "", expectedErr
			},
		}

		call := message.ToolCall{
			ID: "call_3",
			Function: message.FunctionCall{
				Name:      "dummy",
				Arguments: `{}`,
			},
		}

		_, err := appagent.ExecuteToolCall(ctx, call, tl)
		assert.Error(t, err)

		var execErr *appagent.ToolExecutionError
		assert.True(t, errors.As(err, &execErr))
		assert.Equal(t, "dummy", execErr.ToolName)
		assert.Equal(t, expectedErr, execErr.Unwrap())
		assert.Contains(t, err.Error(), "execute tool \"dummy\" failed")
	})

	t.Run("NormalExecution", func(t *testing.T) {
		tl := &dummyTool{
			info: tool.Definition{
				Name: "dummy",
			},
			execute: func(ctx context.Context, args string) (string, error) {
				return "success_result", nil
			},
		}

		call := message.ToolCall{
			ID: "call_4",
			Function: message.FunctionCall{
				Name:      "dummy",
				Arguments: `{}`,
			},
		}

		msg, err := appagent.ExecuteToolCall(ctx, call, tl)
		assert.NoError(t, err)
		assert.Equal(t, "success_result", msg.Content)
		assert.Equal(t, "call_4", msg.ToolCallID)
	})

	t.Run("TimeoutError_PreCheck", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // 立即取消

		call := message.ToolCall{
			ID: "call_5",
			Function: message.FunctionCall{
				Name:      "dummy",
				Arguments: `{}`,
			},
		}

		tl := &dummyTool{
			info: tool.Definition{
				Name: "dummy",
			},
		}

		_, err := appagent.ExecuteToolCall(cancelCtx, call, tl)
		assert.Error(t, err)
		
		var timeoutErr *appagent.TimeoutError
		assert.True(t, errors.As(err, &timeoutErr))
		assert.Equal(t, context.Canceled, timeoutErr.Unwrap())
		assert.Contains(t, err.Error(), "chat timeout or canceled")
	})

	t.Run("TimeoutError_DuringExecute", func(t *testing.T) {
		tl := &dummyTool{
			info: tool.Definition{
				Name: "dummy",
			},
			execute: func(ctx context.Context, args string) (string, error) {
				return "", context.DeadlineExceeded
			},
		}

		call := message.ToolCall{
			ID: "call_6",
			Function: message.FunctionCall{
				Name:      "dummy",
				Arguments: `{}`,
			},
		}

		_, err := appagent.ExecuteToolCall(ctx, call, tl)
		assert.Error(t, err)
		
		var timeoutErr *appagent.TimeoutError
		assert.True(t, errors.As(err, &timeoutErr))
		assert.Equal(t, context.DeadlineExceeded, timeoutErr.Unwrap())
	})
}

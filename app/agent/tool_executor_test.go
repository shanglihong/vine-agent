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

func TestExecuteToolCall_ReturnsOriginalErrors(t *testing.T) {
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
	})

	t.Run("ToolInvalidArgumentError", func(t *testing.T) {
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
				Arguments: `{"name": 123}`, // 类型错误
			},
		}

		_, err := appagent.ExecuteToolCall(ctx, call, tl)
		assert.Error(t, err)

		var invalidArgErr *appagent.ToolInvalidArgumentError
		assert.True(t, errors.As(err, &invalidArgErr))
		assert.Equal(t, "dummy", invalidArgErr.ToolName)
	})

	t.Run("ToolExecutionError", func(t *testing.T) {
		expectedErr := errors.New("db disconnect")
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
		assert.Equal(t, expectedErr, execErr.Err)
	})

	t.Run("TimeoutError", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // 立即取消

		call := message.ToolCall{
			ID: "call_4",
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
	})

	t.Run("ToolConfirmationRequiredError", func(t *testing.T) {
		tl := &dummyTool{
			info: tool.Definition{
				Name:                 "sensitive_tool",
				RequiresConfirmation: true,
			},
		}

		call := message.ToolCall{
			ID: "call_sensitive",
			Function: message.FunctionCall{
				Name:      "sensitive_tool",
				Arguments: `{}`,
			},
		}

		_, err := appagent.ExecuteToolCall(ctx, call, tl)
		assert.Error(t, err)

		var confirmErr *appagent.ToolConfirmationRequiredError
		assert.True(t, errors.As(err, &confirmErr))
		assert.Equal(t, "sensitive_tool", confirmErr.ToolName)
	})
}

func TestConvertToolErrorToMessage(t *testing.T) {
	t.Run("NilError", func(t *testing.T) {
		msg, err := appagent.ConvertToolErrorToMessage("call_1", nil)
		assert.NoError(t, err)
		assert.Equal(t, "", msg.Content)
	})

	t.Run("ToolConfirmationRequiredError_IsReturned", func(t *testing.T) {
		origErr := &appagent.ToolConfirmationRequiredError{ToolName: "sens", ToolCallID: "call_sens"}
		_, err := appagent.ConvertToolErrorToMessage("call_sens", origErr)
		assert.Error(t, err)
		assert.Equal(t, origErr, err)
	})

	t.Run("ToolNotFoundError_IsDigested", func(t *testing.T) {
		origErr := &appagent.ToolNotFoundError{ToolName: "missing"}
		msg, err := appagent.ConvertToolErrorToMessage("call_1", origErr)
		assert.NoError(t, err)
		assert.Contains(t, msg.Content, "error executing tool: tool missing not found")
	})

	t.Run("TimeoutError_IsDigested", func(t *testing.T) {
		origErr := &appagent.TimeoutError{Err: context.DeadlineExceeded}
		msg, err := appagent.ConvertToolErrorToMessage("call_1", origErr)
		assert.NoError(t, err)
		assert.Contains(t, msg.Content, "error executing tool: chat timeout or canceled")
	})
}

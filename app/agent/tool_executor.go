package agent

import (
	"context"
	"errors"
	"fmt"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
)

func ExecuteToolCall(ctx context.Context, call message.ToolCall, t tool.Tool) (message.Message, error) {
	if err := ctx.Err(); err != nil {
		return message.Message{}, &TimeoutError{Err: err}
	}

	if t == nil {
		return message.Message{}, &ToolNotFoundError{ToolName: call.Function.Name}
	}

	// 检查是否需要人工确认并且此次调用未被确认
	if t.Info().RequiresConfirmation && !IsToolCallConfirmed(ctx, call.ID) {
		err := &ToolConfirmationRequiredError{ToolName: t.Info().Name, ToolCallID: call.ID}
		return message.Message{}, err
	}

	// 验证 JSON Schema
	if err := tool.ValidateArguments(t.Info().Parameters, call.Function.Arguments); err != nil {
		return message.Message{}, &ToolInvalidArgumentError{ToolName: t.Info().Name, Err: err}
	}

	res, err := t.Execute(ctx, call.Function.Arguments)
	if err != nil {
		if ctx.Err() != nil {
			return message.Message{}, &TimeoutError{Err: ctx.Err()}
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return message.Message{}, &TimeoutError{Err: err}
		}
		return message.Message{}, &ToolExecutionError{ToolName: t.Info().Name, Err: err}
	}

	return message.NewToolMessage(call.ID, res), nil
}


// ConvertToolErrorToMessage 将工具执行返回的 Go 错误分类处理。
// 如果是需要阻断的人工确认错误，则原样返回 Go error，由外部处理控制流；
// 如果是普通业务/系统错误（如参数错误、执行错误、超时等），则将其包装并消化为含有报错回显文案的 ToolMessage，并返回 nil error。
func ConvertToolErrorToMessage(callID string, err error) (message.Message, error) {
	if err == nil {
		return message.Message{}, nil
	}

	// 1. 阻断型控制错误：人工确认挂起
	var confirmErr *ToolConfirmationRequiredError
	if errors.As(err, &confirmErr) {
		return message.Message{}, confirmErr
	}

	// 2. 业务性与系统级错误：转换为带有报错回显的 ToolResult Message，且返回 nil error
	var msgContent string
	switch e := err.(type) {
	case *ToolNotFoundError:
		msgContent = fmt.Sprintf("error executing tool: %s", e.Error())
	case *ToolInvalidArgumentError:
		msgContent = fmt.Sprintf("error executing tool: %s", e.Error())
	case *ToolExecutionError:
		msgContent = fmt.Sprintf("error executing tool: %s", e.Error())
	case *TimeoutError:
		msgContent = fmt.Sprintf("error executing tool: %s", e.Error())
	default:
		// 降级兜底
		msgContent = fmt.Sprintf("error executing tool: %s", e.Error())
	}

	return message.NewToolMessage(callID, msgContent), nil
}

package agent

import (
	"context"
	"errors"
	"fmt"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"

	"github.com/xeipuuv/gojsonschema"
)

func ExecuteToolCall(ctx context.Context, call message.ToolCall, tool tool.Tool) (message.Message, error) {
	if err := ctx.Err(); err != nil {
		return message.Message{}, &TimeoutError{Err: err}
	}

	if tool == nil {
		return message.Message{}, &ToolNotFoundError{ToolName: call.Function.Name}
	}

	// 检查是否需要人工确认并且此次调用未被确认
	if tool.Info().RequiresConfirmation && !IsToolCallConfirmed(ctx, call.ID) {
		err := &ToolConfirmationRequiredError{ToolName: tool.Info().Name, ToolCallID: call.ID}
		return message.Message{}, err
	}

	// 验证 JSON Schema
	if err := validateArguments(tool.Info().Parameters, call.Function.Arguments); err != nil {
		return message.Message{}, &ToolInvalidArgumentError{ToolName: tool.Info().Name, Err: err}
	}

	res, err := tool.Execute(ctx, call.Function.Arguments)
	if err != nil {
		if ctx.Err() != nil {
			return message.Message{}, &TimeoutError{Err: ctx.Err()}
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return message.Message{}, &TimeoutError{Err: err}
		}
		return message.Message{}, &ToolExecutionError{ToolName: tool.Info().Name, Err: err}
	}

	return message.NewToolMessage(call.ID, res), nil
}

// validateArguments 校验入参是否符合 JSON Schema 规范
func validateArguments(schema any, args string) error {
	if schema == nil {
		return nil
	}

	schemaLoader := gojsonschema.NewGoLoader(schema)
	documentLoader := gojsonschema.NewStringLoader(args)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		return fmt.Errorf("validation failed: %v", result.Errors())
	}
	return nil
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

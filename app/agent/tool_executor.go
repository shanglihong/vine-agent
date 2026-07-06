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
		err := &ToolNotFoundError{ToolName: call.Function.Name}
		return message.NewToolMessage(call.ID, fmt.Sprintf("error executing tool: %s", err.Error())), err
	}

	// 验证 JSON Schema
	if err := validateArguments(tool.Info().Parameters, call.Function.Arguments); err != nil {
		wrapErr := &ToolInvalidArgumentError{ToolName: tool.Info().Name, Err: err}
		return message.NewToolMessage(call.ID, fmt.Sprintf("error executing tool: %s", wrapErr.Error())), wrapErr
	}

	res, err := tool.Execute(ctx, call.Function.Arguments)
	if err != nil {
		if ctx.Err() != nil {
			return message.Message{}, &TimeoutError{Err: ctx.Err()}
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return message.Message{}, &TimeoutError{Err: err}
		}
		wrapErr := &ToolExecutionError{ToolName: tool.Info().Name, Err: err}
		return message.NewToolMessage(call.ID, fmt.Sprintf("error executing tool: %s", wrapErr.Error())), wrapErr
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

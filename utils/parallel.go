package utils

import (
	"context"
	"sync"
)

// ParallelMap 并发地将 inputs 中的每个元素 I 转换为 O。
// 默认不设并发度限制，且在任一任务出错时，默认快速失败并取消其他执行中的任务。
func ParallelMap[I any, O any](ctx context.Context, inputs []I, mapFunc func(context.Context, I) (O, error)) ([]O, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	outputs := make([]O, len(inputs))
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	for i, input := range inputs {
		i, input := i, input

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			out, err := mapFunc(ctx, input)
			if err != nil {
				errOnce.Do(func() {
					firstErr = err
					cancel() // 默认 failFast，取消 context 使得其他协程能感知并快速退出
				})
				return
			}
			outputs[i] = out
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return outputs, nil
}

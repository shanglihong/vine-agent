package utils

import (
	"context"
	"sync"
)

type ParallelResult[I any, O any] struct {
	I     I
	O     O
	Error error
}

// ParallelResults 包装 ParallelResult 切片，用于提供结果处理的便捷方法
type ParallelResults[I any, O any] []ParallelResult[I, O]

// Split 将结果分为成功列表和失败列表。
// 成功的列表只包含 error 为空的 ParallelResult，失败的列表只包含 error 不为空的 ParallelResult。
func (r ParallelResults[I, O]) Split() (success ParallelResults[I, O], failure ParallelResults[I, O]) {
	for _, res := range r {
		if res.Error == nil {
			success = append(success, res)
		} else {
			failure = append(failure, res)
		}
	}
	return
}

// Successes 返回所有 error 为空的 ParallelResult 列表。
func (r ParallelResults[I, O]) Successes() ParallelResults[I, O] {
	var succs ParallelResults[I, O]
	for _, res := range r {
		if res.Error == nil {
			succs = append(succs, res)
		}
	}
	return succs
}

// Failures 返回所有 error 不为空的 ParallelResult 列表。
func (r ParallelResults[I, O]) Failures() ParallelResults[I, O] {
	var fails ParallelResults[I, O]
	for _, res := range r {
		if res.Error != nil {
			fails = append(fails, res)
		}
	}
	return fails
}

// Option 定义并行执行参数的配置项。
type Option func(*parallelOptions)

type parallelOptions struct {
	failFast bool
}

// WithFailFast 允许配置是否在任何任务出错时立即取消其余并行任务（默认为 true）。
func WithFailFast(failFast bool) Option {
	return func(o *parallelOptions) {
		o.failFast = failFast
	}
}

// ParallelMap 并发地将 inputs 中的每个元素 I 转换为 O。
// 默认不设并发度限制，且在任一任务出错时，默认快速失败并取消其他执行中的任务。
// 返回的切片长度与 inputs 一致，包含每个 input 的输入、输出与发生的错误。
func ParallelMap[I any, O any](ctx context.Context, inputs []I, mapFunc func(context.Context, I) (O, error), opts ...Option) ParallelResults[I, O] {
	if len(inputs) == 0 {
		return nil
	}

	opt := parallelOptions{
		failFast: true, // 默认开启快速失败
	}
	for _, o := range opts {
		o(&opt)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(ParallelResults[I, O], len(inputs))
	for i, input := range inputs {
		results[i].I = input
	}

	var wg sync.WaitGroup
	var errOnce sync.Once

	for i, input := range inputs {
		i, input := i, input

		select {
		case <-ctx.Done():
			results[i].Error = ctx.Err()
			continue
		default:
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			out, err := mapFunc(ctx, input)
			if err != nil {
				if opt.failFast {
					errOnce.Do(func() {
						cancel() // 默认 failFast，取消 context 使得其他协程能感知并快速退出
					})
				}
				results[i].Error = err
				return
			}
			results[i].O = out
		}()
	}

	wg.Wait()
	return results
}

package utils_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"vine-agent/utils"

	"github.com/stretchr/testify/assert"
)

func TestParallelMap_Success(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}

	// 模拟并发转换逻辑，较长输入休眠更久，以测试结果切片顺序是否会错乱
	outputs := utils.ParallelMap(context.Background(), inputs, func(ctx context.Context, input int) (int, error) {
		sleepMs := time.Duration(10-input) * time.Millisecond
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(sleepMs):
		}
		return input * 2, nil
	})

	assert.Len(t, outputs, 5)
	for i, input := range inputs {
		assert.Equal(t, input, outputs[i].I)
		assert.Equal(t, input*2, outputs[i].O)
		assert.NoError(t, outputs[i].Error)
	}
}

func TestParallelMap_Empty(t *testing.T) {
	outputs := utils.ParallelMap(context.Background(), []string{}, func(ctx context.Context, input string) (string, error) {
		return input, nil
	})
	assert.Nil(t, outputs)
}

func TestParallelMap_FailFast(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}
	targetErr := errors.New("error on 3")

	var canceledCount int
	var mu sync.Mutex

	outputs := utils.ParallelMap(context.Background(), inputs, func(ctx context.Context, input int) (int, error) {
		if input == 3 {
			// 让 3 号任务迅速出错返回
			return 0, targetErr
		}

		// 其他任务等待较长时间，观察 context 是否会被取消
		sleepMs := 100 * time.Millisecond
		select {
		case <-ctx.Done():
			mu.Lock()
			canceledCount++
			mu.Unlock()
			return 0, ctx.Err()
		case <-time.After(sleepMs):
		}
		return input, nil
	})

	assert.Len(t, outputs, 5)
	assert.ErrorIs(t, outputs[2].Error, targetErr)
	assert.Equal(t, 3, outputs[2].I)

	var hasCanceled bool
	for _, res := range outputs {
		if errors.Is(res.Error, context.Canceled) {
			hasCanceled = true
		}
	}
	assert.True(t, hasCanceled, "should contain canceled elements due to failFast")

	// 等待一小会儿确保所有 goroutine 都退出了
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	// 除 3 以外的 4 个任务应该因为 context canceled 而被取消了大部分
	assert.Greater(t, canceledCount, 0)
}

func TestParallelMap_DisableFailFast(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}
	targetErr := errors.New("error on 3")

	var finishedCount int
	var mu sync.Mutex

	outputs := utils.ParallelMap(context.Background(), inputs, func(ctx context.Context, input int) (int, error) {
		if input == 3 {
			// 让 3 号任务迅速出错返回
			return 0, targetErr
		}

		// 其他任务等待较长时间，但因为禁用了 failFast，应该都能正常跑完而不会被 cancel
		sleepMs := 30 * time.Millisecond
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(sleepMs):
			mu.Lock()
			finishedCount++
			mu.Unlock()
		}
		return input * 2, nil
	}, utils.WithFailFast(false)) // 禁用 failFast

	assert.Len(t, outputs, 5)

	// 3 号任务返回错误
	assert.ErrorIs(t, outputs[2].Error, targetErr)

	// 其他 4 个任务应该全部顺利完成
	assert.Equal(t, 4, finishedCount)
	for i, res := range outputs {
		if i == 2 {
			continue
		}
		assert.NoError(t, res.Error)
		assert.Equal(t, res.I*2, res.O)
	}
}

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
	outputs, err := utils.ParallelMap(context.Background(), inputs, func(ctx context.Context, input int) (int, error) {
		sleepMs := time.Duration(10-input) * time.Millisecond
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(sleepMs):
		}
		return input * 2, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []int{2, 4, 6, 8, 10}, outputs)
}

func TestParallelMap_Empty(t *testing.T) {
	outputs, err := utils.ParallelMap(context.Background(), []string{}, func(ctx context.Context, input string) (string, error) {
		return input, nil
	})
	assert.NoError(t, err)
	assert.Nil(t, outputs)
}

func TestParallelMap_FailFast(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}
	targetErr := errors.New("error on 3")

	var canceledCount int
	var mu sync.Mutex

	_, err := utils.ParallelMap(context.Background(), inputs, func(ctx context.Context, input int) (int, error) {
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

	assert.ErrorIs(t, err, targetErr)

	// 等待一小会儿确保所有 goroutine 都退出了
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	// 除 3 以外的 4 个任务应该因为 context canceled 而被取消了大部分
	assert.Greater(t, canceledCount, 0)
}

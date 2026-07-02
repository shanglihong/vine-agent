package file_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"vine-agent/domain/memory/profile"
	"vine-agent/infra/persistence/file"
)

func TestFileProfileRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("GetByUserID returns nil when both files do not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := file.NewFileProfileRepository(tempDir)

		prof, err := repo.GetByUserID(ctx, "user_nonexistent")
		assert.NoError(t, err)
		assert.Nil(t, prof)
	})

	t.Run("Save and GetByUserID successful flow", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := file.NewFileProfileRepository(tempDir)

		userID := "user_123"
		prof := profile.NewProfile(userID)
		prof.Preferences = []string{"喜欢写 Go", "讨厌加班"}
		prof.Facts = []string{"现居上海", "职业为架构师"}

		// 保存数据
		err := repo.Save(ctx, prof)
		assert.NoError(t, err)

		// 检查在 tempDir/user_123 目录下是否生成了对应的文件
		userDir := filepath.Join(tempDir, userID)
		assert.DirExists(t, userDir)
		assert.FileExists(t, filepath.Join(userDir, "preferences.md"))
		assert.FileExists(t, filepath.Join(userDir, "facts.md"))

		// 从磁盘读取回来
		loadedProf, err := repo.GetByUserID(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, loadedProf)
		assert.Equal(t, userID, loadedProf.UserID)
		assert.Equal(t, []string{"喜欢写 Go", "讨厌加班"}, loadedProf.Preferences)
		assert.Equal(t, []string{"现居上海", "职业为架构师"}, loadedProf.Facts)
	})

	t.Run("Read Markdown files with arbitrary lists format", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := file.NewFileProfileRepository(tempDir)

		userID := "user_special"
		userDir := filepath.Join(tempDir, userID)
		err := os.MkdirAll(userDir, 0755)
		assert.NoError(t, err)

		// 模拟写入带有多余空格、星号与加号前缀的非标准 Markdown 列表文件
		prefContent := `
  *   学习 Rust 语言
  + 喜欢喝咖啡
-  日常爱听音乐  
`
		factContent := `
- 是一名技术总监
*   年龄 30+ 岁
`
		err = os.WriteFile(filepath.Join(userDir, "preferences.md"), []byte(prefContent), 0644)
		assert.NoError(t, err)

		err = os.WriteFile(filepath.Join(userDir, "facts.md"), []byte(factContent), 0644)
		assert.NoError(t, err)

		// 读取验证
		loaded, err := repo.GetByUserID(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, loaded)

		// 校验去除 Markdown 前缀与 Trim 空格后的结果
		expectedPrefs := []string{"学习 Rust 语言", "喜欢喝咖啡", "日常爱听音乐"}
		expectedFacts := []string{"是一名技术总监", "年龄 30+ 岁"}

		assert.Equal(t, expectedPrefs, loaded.Preferences)
		assert.Equal(t, expectedFacts, loaded.Facts)
	})

	t.Run("Concurrently read and write profiles", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := file.NewFileProfileRepository(tempDir)
		userID := "concurrent_user"

		var wg sync.WaitGroup
		concurrency := 10

		// 并发写
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				prof := profile.NewProfile(userID)
				prof.Preferences = []string{"偏好项"}
				prof.Facts = []string{"事实项"}
				_ = repo.Save(ctx, prof)
			}(i)
		}

		// 并发读
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = repo.GetByUserID(ctx, userID)
			}()
		}

		wg.Wait()

		// 最终读取验证
		loaded, err := repo.GetByUserID(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, []string{"偏好项"}, loaded.Preferences)
		assert.Equal(t, []string{"事实项"}, loaded.Facts)
	})
}

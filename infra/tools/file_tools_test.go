package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vine-agent/app/agent"
	"vine-agent/infra/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDirTool_Execute(t *testing.T) {
	tempDir := t.TempDir()

	// 创建测试文件和子目录
	err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("hello"), 0644)
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	require.NoError(t, err)

	toolInstance := tools.NewListDirTool()

	// 注入项目根路径到 Context
	ctx := agent.WithProjectPath(context.Background(), tempDir)

	t.Run("未注入项目路径时报错", func(t *testing.T) {
		_, err := toolInstance.Execute(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project path not found in context")
	})

	t.Run("列出项目根目录内容", func(t *testing.T) {
		args, err := json.Marshal(map[string]string{"path": "."})
		require.NoError(t, err)

		res, err := toolInstance.Execute(ctx, string(args))
		assert.NoError(t, err)

		type FileItem struct {
			Name    string `json:"name"`
			Size    int64  `json:"size"`
			IsDir   bool   `json:"is_dir"`
			ModTime string `json:"mod_time"`
		}

		var items []FileItem
		err = json.Unmarshal([]byte(res), &items)
		assert.NoError(t, err)

		assert.Len(t, items, 2)
		names := []string{items[0].Name, items[1].Name}
		assert.Contains(t, names, "file1.txt")
		assert.Contains(t, names, "subdir")

		// 验证属性
		for _, item := range items {
			if item.Name == "file1.txt" {
				assert.Equal(t, int64(5), item.Size)
				assert.False(t, item.IsDir)
			} else {
				assert.True(t, item.IsDir)
			}
			assert.NotEmpty(t, item.ModTime)
		}
	})

	t.Run("目录路径不存在时报错并不暴露绝对路径", func(t *testing.T) {
		args, err := json.Marshal(map[string]string{"path": "non_existent"})
		require.NoError(t, err)

		_, err = toolInstance.Execute(ctx, string(args))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "读取目录失败")
		// 校验错误信息是否对绝对路径进行了处理，不含有 tempDir 部分
		assert.NotContains(t, err.Error(), tempDir)
		assert.Contains(t, err.Error(), "non_existent")
	})
}

func TestReadFilesTool_Execute(t *testing.T) {
	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content 1"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("content 2"), 0644)
	require.NoError(t, err)

	toolInstance := tools.NewReadFilesTool()
	ctx := agent.WithProjectPath(context.Background(), tempDir)

	t.Run("成功读取多个存在的文件", func(t *testing.T) {
		args, err := json.Marshal(map[string][]string{
			"paths": {"file1.txt", "file2.txt"},
		})
		require.NoError(t, err)

		res, err := toolInstance.Execute(ctx, string(args))
		assert.NoError(t, err)

		type FileResult struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Error   string `json:"error,omitempty"`
		}

		var results []FileResult
		err = json.Unmarshal([]byte(res), &results)
		assert.NoError(t, err)

		assert.Len(t, results, 2)
		assert.Equal(t, "file1.txt", results[0].Path)
		assert.Equal(t, "content 1", results[0].Content)
		assert.Empty(t, results[0].Error)

		assert.Equal(t, "file2.txt", results[1].Path)
		assert.Equal(t, "content 2", results[1].Content)
		assert.Empty(t, results[1].Error)
	})

	t.Run("部分文件不存在时容错输出错误信息且不暴露绝对路径", func(t *testing.T) {
		args, err := json.Marshal(map[string][]string{
			"paths": {"file1.txt", "non_existent.txt"},
		})
		require.NoError(t, err)

		res, err := toolInstance.Execute(ctx, string(args))
		assert.NoError(t, err)

		type FileResult struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Error   string `json:"error,omitempty"`
		}

		var results []FileResult
		err = json.Unmarshal([]byte(res), &results)
		assert.NoError(t, err)

		assert.Len(t, results, 2)
		assert.Equal(t, "file1.txt", results[0].Path)
		assert.Equal(t, "content 1", results[0].Content)
		assert.Empty(t, results[0].Error)

		assert.Equal(t, "non_existent.txt", results[1].Path)
		assert.Empty(t, results[1].Content)
		assert.NotEmpty(t, results[1].Error)
		assert.NotContains(t, results[1].Error, tempDir)
	})
}

func TestWriteFileTool_Execute(t *testing.T) {
	tempDir := t.TempDir()

	toolInstance := tools.NewWriteFileTool()
	ctx := agent.WithProjectPath(context.Background(), tempDir)

	t.Run("成功写入新文件并自动创建父目录且不暴露绝对路径", func(t *testing.T) {
		args, err := json.Marshal(map[string]string{
			"path":    "subdir/test.txt",
			"content": "hello world",
		})
		require.NoError(t, err)

		res, err := toolInstance.Execute(ctx, string(args))
		assert.NoError(t, err)
		assert.Contains(t, res, "Successfully wrote")
		// 校验返回信息不包含绝对路径
		assert.NotContains(t, res, tempDir)
		assert.Contains(t, res, "subdir/test.txt")

		// 验证文件存在且内容正确
		data, err := os.ReadFile(filepath.Join(tempDir, "subdir", "test.txt"))
		assert.NoError(t, err)
		assert.Equal(t, "hello world", string(data))
	})

	t.Run("成功覆盖写入已存在的文件", func(t *testing.T) {
		args, err := json.Marshal(map[string]string{
			"path":    "subdir/test.txt",
			"content": "new content",
		})
		require.NoError(t, err)

		_, err = toolInstance.Execute(ctx, string(args))
		assert.NoError(t, err)

		// 验证文件被覆盖
		data, err := os.ReadFile(filepath.Join(tempDir, "subdir", "test.txt"))
		assert.NoError(t, err)
		assert.Equal(t, "new content", string(data))
	})

	t.Run("文件路径为空时报错", func(t *testing.T) {
		args, err := json.Marshal(map[string]string{
			"path":    "",
			"content": "content",
		})
		require.NoError(t, err)

		_, err = toolInstance.Execute(ctx, string(args))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文件路径不能为空")
	})
}

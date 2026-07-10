package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vine-agent/app/agent"
	"vine-agent/domain/tool"
)

// resolvePath 将输入路径拼接到项目工作区根目录下
func resolvePath(projectPath, inputPath string) string {
	return filepath.Join(projectPath, inputPath)
}

// makeRelative 转换为相对于项目工作空间的相对路径，以对大模型隐蔽物理绝对路径
func makeRelative(projectPath, path string) string {
	rel, err := filepath.Rel(projectPath, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}

// sanitizeError 处理错误消息，消除其中包含的项目物理绝对路径信息
func sanitizeError(projectPath, path string, err error) error {
	if err == nil {
		return nil
	}
	errStr := err.Error()
	absPath := resolvePath(projectPath, path)
	relPath := makeRelative(projectPath, absPath)

	errStr = strings.ReplaceAll(errStr, absPath, relPath)
	cleanProjPath := filepath.Clean(projectPath)
	errStr = strings.ReplaceAll(errStr, cleanProjPath, "")
	return fmt.Errorf("%s", errStr)
}

// ListDirTool 列出指定目录下的子文件与子目录
type ListDirTool struct{}

// NewListDirTool 创建 ListDirTool 实例
func NewListDirTool() tool.Tool {
	return &ListDirTool{}
}

// Info 返回工具定义元数据
func (t *ListDirTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "list_dir",
		Description: "列出指定文件夹目录下的所有子文件和子文件夹列表。返回结果为 JSON 格式 of 列表，包含 name, size, is_dir, mod_time 字段。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "要查询的目录路径，可使用相对路径或绝对路径。如果为空，则默认列出当前工作目录内容。",
				},
			},
		},
		RequiresConfirmation: false,
	}
}

// Execute 执行列出目录的逻辑
func (t *ListDirTool) Execute(ctx context.Context, args string) (string, error) {
	projectPath, ok := agent.GetProjectPath(ctx)
	if !ok || projectPath == "" {
		return "", fmt.Errorf("project path not found in context")
	}

	var params struct {
		Path string `json:"path"`
	}
	if args != "" && args != "{}" {
		_ = json.Unmarshal([]byte(args), &params)
	}

	path := strings.TrimSpace(params.Path)
	if path == "" {
		path = "."
	}

	targetDir := resolvePath(projectPath, path)
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %w", sanitizeError(projectPath, path, err))
	}

	type FileItem struct {
		Name    string `json:"name"`
		Size    int64  `json:"size"`
		IsDir   bool   `json:"is_dir"`
		ModTime string `json:"mod_time"`
	}

	var items []FileItem
	for _, entry := range entries {
		var size int64
		var modTime string

		info, err := entry.Info()
		if err == nil {
			size = info.Size()
			modTime = info.ModTime().Format(time.RFC3339)
		}

		items = append(items, FileItem{
			Name:    entry.Name(),
			Size:    size,
			IsDir:   entry.IsDir(),
			ModTime: modTime,
		})
	}

	resBytes, err := json.Marshal(items)
	if err != nil {
		return "", fmt.Errorf("序列化目录内容失败: %w", err)
	}
	return string(resBytes), nil
}

// ReadFilesTool 读取多个文件内容
type ReadFilesTool struct{}

// NewReadFilesTool 创建 ReadFilesTool 实例
func NewReadFilesTool() tool.Tool {
	return &ReadFilesTool{}
}

// Info 返回工具定义元数据
func (t *ReadFilesTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "read_files",
		Description: "读取指定的文件列表内容。返回一个 JSON 数组，包含每个文件路径对应的内容或读取失败的错误信息。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"paths": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "需要读取的文件物理路径列表（支持相对或绝对路径）",
				},
			},
			"required": []any{"paths"},
		},
		RequiresConfirmation: false,
	}
}

// Execute 执行读取多个文件的逻辑
func (t *ReadFilesTool) Execute(ctx context.Context, args string) (string, error) {
	projectPath, ok := agent.GetProjectPath(ctx)
	if !ok || projectPath == "" {
		return "", fmt.Errorf("project path not found in context")
	}

	var params struct {
		Paths []string `json:"paths"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	type FileResult struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Error   string `json:"error,omitempty"`
	}

	var results []FileResult
	for _, p := range params.Paths {
		targetPath := resolvePath(projectPath, p)
		content, err := os.ReadFile(targetPath)
		if err != nil {
			results = append(results, FileResult{
				Path:  makeRelative(projectPath, targetPath),
				Error: fmt.Errorf("读取文件失败: %w", sanitizeError(projectPath, p, err)).Error(),
			})
		} else {
			results = append(results, FileResult{
				Path:    makeRelative(projectPath, targetPath),
				Content: string(content),
			})
		}
	}

	resBytes, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("序列化文件读取结果失败: %w", err)
	}
	return string(resBytes), nil
}

// WriteFileTool 写入文件内容，覆盖已存在文件
type WriteFileTool struct{}

// NewWriteFileTool 创建 WriteFileTool 实例
func NewWriteFileTool() tool.Tool {
	return &WriteFileTool{}
}

// Info 返回工具定义元数据
func (t *WriteFileTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "write_file",
		Description: "向指定路径写入文件内容。如果文件不存在则自动创建，如果已存在则直接覆盖。会自动递归创建不存在的父目录。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "要写入的文件路径（支持相对或绝对路径）",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "写入文件的纯文本内容",
				},
			},
			"required": []any{"path", "content"},
		},
		RequiresConfirmation: false,
	}
}

// Execute 执行写入文件的逻辑
func (t *WriteFileTool) Execute(ctx context.Context, args string) (string, error) {
	projectPath, ok := agent.GetProjectPath(ctx)
	if !ok || projectPath == "" {
		return "", fmt.Errorf("project path not found in context")
	}

	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	path := strings.TrimSpace(params.Path)
	if path == "" {
		return "", fmt.Errorf("文件路径不能为空")
	}

	targetPath := resolvePath(projectPath, path)

	// 自动创建父目录
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	if err := os.WriteFile(targetPath, []byte(params.Content), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", sanitizeError(projectPath, path, err))
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(params.Content), makeRelative(projectPath, targetPath)), nil
}

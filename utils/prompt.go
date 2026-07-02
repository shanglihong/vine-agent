package utils

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// RenderPromptFile 读取项目根目录下指定相对路径的模板文件并执行渲染
func RenderPromptFile(relPath string, data interface{}) (string, error) {
	root := FindProjectRoot()
	if root == "" {
		return "", fmt.Errorf("project root not found")
	}

	path := filepath.Join(root, relPath)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s: %w", path, err)
	}

	tmplName := filepath.Base(relPath)
	tmpl, err := template.New(tmplName).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template: %w", err)
	}

	return buf.String(), nil
}

package file

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vine-agent/domain/memory/profile"
)

type fileProfileRepository struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileProfileRepository 创建一个基于本地 Markdown 文件的 Profile 仓储实现
func NewFileProfileRepository(baseDir string) profile.ProfileRepository {
	return &fileProfileRepository{
		baseDir: baseDir,
	}
}

// GetByUserID 获取指定用户的记忆画像。如果偏好和事实文件均不存在，返回 (nil, nil)
func (r *fileProfileRepository) GetByUserID(ctx context.Context, userID string) (*profile.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	prefPath := filepath.Join(r.baseDir, userID, "preferences.md")
	factPath := filepath.Join(r.baseDir, userID, "facts.md")

	prefExist := r.fileExists(prefPath)
	factExist := r.fileExists(factPath)

	// 如果两个文件都不存在，按照契约返回 (nil, nil)
	if !prefExist && !factExist {
		return nil, nil
	}

	prof := profile.NewProfile(userID)

	if prefExist {
		prefs, err := r.readListFile(prefPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read preferences file: %w", err)
		}
		prof.Preferences = prefs
	}

	if factExist {
		facts, err := r.readListFile(factPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read facts file: %w", err)
		}
		prof.Facts = facts
	}

	// 取两个文件的更新时间中较新的一位，或者若无法获取则用当前时间
	var lastUpdated time.Time
	if prefExist {
		if info, err := os.Stat(prefPath); err == nil {
			lastUpdated = info.ModTime()
		}
	}
	if factExist {
		if info, err := os.Stat(factPath); err == nil && info.ModTime().After(lastUpdated) {
			lastUpdated = info.ModTime()
		}
	}
	if lastUpdated.IsZero() {
		lastUpdated = time.Now()
	}
	prof.UpdatedAt = lastUpdated

	return prof, nil
}

// Save 保存用户的记忆画像至本地两个 Markdown 文件（偏好和事实分别存储）
func (r *fileProfileRepository) Save(ctx context.Context, prof *profile.Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	userDir := filepath.Join(r.baseDir, prof.UserID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return fmt.Errorf("failed to create user directory: %w", err)
	}

	prefPath := filepath.Join(userDir, "preferences.md")
	factPath := filepath.Join(userDir, "facts.md")

	// 分别写入偏好和事实文件
	if err := r.writeListFile(prefPath, prof.Preferences); err != nil {
		return fmt.Errorf("failed to save preferences file: %w", err)
	}

	if err := r.writeListFile(factPath, prof.Facts); err != nil {
		return fmt.Errorf("failed to save facts file: %w", err)
	}

	return nil
}

// fileExists 检查文件是否存在
func (r *fileProfileRepository) fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && !info.IsDir()
}

// readListFile 读取 Markdown 列表文件并将每一项还原为字符串列表
func (r *fileProfileRepository) readListFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var items []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 剥离 Markdown 列表符号: 支持 "-" 或 "*" 或 "+"
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "+") {
			line = strings.TrimLeft(line, "-*+ ")
		}
		line = strings.TrimSpace(line)
		if line != "" {
			items = append(items, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// writeListFile 将字符串列表格式化写入 Markdown 列表文件
func (r *fileProfileRepository) writeListFile(path string, items []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		// 统一写入为 "-" 列表形式
		if _, err := writer.WriteString(fmt.Sprintf("- %s\n", item)); err != nil {
			return err
		}
	}

	return writer.Flush()
}

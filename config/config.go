package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"vine-agent/utils"
)

// Config 统一管理系统各模块的配置项与路径
type Config struct {
	// ProfileDir 长期记忆画像（偏好与事实）存放的 Markdown 文件目录
	ProfileDir string `json:"profile_dir" yaml:"profile_dir"`
	// SQLiteDBPath SQLite 数据库（存储会话和检索索引）的物理文件路径
	SQLiteDBPath string `json:"sqlite_db_path" yaml:"sqlite_db_path"`
}

// DefaultConfig 构造并返回 Config 配置实例。
// 如果环境变量 APP_ENV 为 "dev"/"development" 或 VINE_DEBUG 为 "true"，将数据保存在项目根目录下的 data/ 目录；
// 否则为默认生产部署环境，将数据保存在用户主目录下的隐藏文件夹 ~/.vine-agent/ 中。
func DefaultConfig() *Config {
	env := strings.ToLower(os.Getenv("APP_ENV"))
	isDebug := os.Getenv("VINE_DEBUG") == "true" || env == "dev" || env == "development"

	if isDebug {
		root := utils.FindProjectRoot()
		if root != "" {
			baseDir := filepath.Join(root, "data")
			return &Config{
				ProfileDir:   filepath.Join(baseDir, "profile"),
				SQLiteDBPath: filepath.Join(baseDir, "db", "memory.db"),
			}
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "." // 无法获取主目录时，降级退化到当前工作目录
	}

	baseDir := filepath.Join(home, ".vine-agent")
	return &Config{
		ProfileDir:   filepath.Join(baseDir, "profile"),
		SQLiteDBPath: filepath.Join(baseDir, "db", "memory.db"),
	}
}

// LoadConfigFromFile 从指定路径加载并解析 YAML 配置文件。
// 它会首先基于 DefaultConfig() 初始化默认值，然后通过 YAML 覆盖文件里显式指定的字段。
func LoadConfigFromFile(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml config: %w", err)
	}

	return cfg, nil
}

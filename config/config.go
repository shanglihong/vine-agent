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
	Server struct {
		// Port 接口服务监听的端口号，例如 ":8080" 或 "8080"
		Port string `json:"port" yaml:"port"`
	} `json:"server" yaml:"server"`

	Storage struct {
		// ProfileDir 长期记忆画像（偏好与事实）存放的 Markdown 文件目录
		ProfileDir string `json:"profile_dir" yaml:"profile_dir"`
		// SQLiteDBPath SQLite 数据库（存储会话和检索索引）的物理文件路径
		SQLiteDBPath string `json:"sqlite_db_path" yaml:"sqlite_db_path"`
		// ProjectRootDir 项目（Project）物理工作空间的根目录
		ProjectRootDir string `json:"project_root_dir" yaml:"project_root_dir"`
	} `json:"storage" yaml:"storage"`
}

// DefaultConfig 构造并返回 Config 配置实例。
// 如果环境变量 APP_ENV 为 "dev"/"development" 或 VINE_DEBUG 为 "true"，将数据保存在项目根目录下的 data/ 目录；
// 否则为默认生产部署环境，将数据保存在用户主目录下的隐藏文件夹 ~/.vine-agent/ 中。
func DefaultConfig() *Config {
	var cfg Config
	cfg.Server.Port = ":8080"
	cfg.Storage.ProfileDir = filepath.Join(utils.FindProjectRoot(), "data", "profile")
	cfg.Storage.SQLiteDBPath = filepath.Join(utils.FindProjectRoot(), "data", "db", "memory.db")
	cfg.Storage.ProjectRootDir = filepath.Join(utils.FindProjectRoot(), "project")
	return &cfg
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

	// 统一展开波浪号 ~ 为用户主目录绝对路径
	cfg.Storage.ProfileDir = expandPath(cfg.Storage.ProfileDir)
	cfg.Storage.SQLiteDBPath = expandPath(cfg.Storage.SQLiteDBPath)
	cfg.Storage.ProjectRootDir = expandPath(cfg.Storage.ProjectRootDir)

	return cfg, nil
}

// expandPath 将路径中开头的波浪号 ~ 展开为实际的用户主目录绝对路径
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		if strings.HasPrefix(path, "~/") {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func LoadConfig() *Config {
	cfg, _ := LoadConfigFromFile("config.yaml")
	env := strings.ToLower(os.Getenv("APP_ENV"))
	if env == "dev" {
		return DefaultConfig()
	}
	return cfg
}

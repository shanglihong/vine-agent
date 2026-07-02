package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"vine-agent/utils"

	_ "modernc.org/sqlite"
)

var (
	memoryDBOnce sync.Once
	memoryDBConn *sql.DB
	memoryDBErr  error
)

// openDatabase 打开指定路径的 SQLite 数据库，并确保其所在的父目录已创建
func openDatabase(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}
	return db, nil
}

// MemoryDBPath 返回项目 SQLite memory 数据库路径
func MemoryDBPath() string {
	root := utils.FindProjectRoot()
	if root != "" {
		return filepath.Join(root, "data", "memory")
	}
	return "data/memory"
}

// getMemoryDB 获取或初始化共享的 SQLite 数据库连接，接受 dbPath 作为传入参数
func getMemoryDB(dbPath string) (*sql.DB, error) {
	memoryDBOnce.Do(func() {
		db, err := openDatabase(dbPath)
		if err != nil {
			memoryDBErr = err
			return
		}

		memoryDBConn = db
	})
	return memoryDBConn, memoryDBErr
}

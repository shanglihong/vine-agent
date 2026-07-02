package utils

import (
	"os"
	"path/filepath"
)

// FindProjectRoot 向上递归寻找 go.mod 文件，以确定项目的根目录
func FindProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

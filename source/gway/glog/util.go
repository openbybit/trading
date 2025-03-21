package glog

import (
	"fmt"
	"os"
	"path/filepath"
)

func mkdirAll(path string) error {
	if info, err := os.Stat(path); os.IsNotExist(err) || !info.IsDir() {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}

// getProcessName 获取进程名
func getProcessName() string {
	return filepath.Base(os.Args[0])
}

// getProjectName 获取项目名,用于自动拼接日志目录
func getProjectName() string {
	if n := os.Getenv("GWAY_PROJECT_NAME"); n != "" {
		return n
	}

	if n := os.Getenv("MY_PROJECT_ENV_NAME"); n != "" {
		return n
	}

	return "gway"
}

// getDefaultFilename 返回约定的项目名
func getDefaultFilename() string {
	return fmt.Sprintf("/data/logs/%s/%s.log", getProjectName(), getProcessName())
}

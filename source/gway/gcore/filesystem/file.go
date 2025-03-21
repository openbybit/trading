package filesystem

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func ReadFile(file string) ([]byte, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func OpenFile(file string) (*os.File, error) {
	path := filepath.Dir(file)
	if err := MkdirAll(path); err != nil {
		return nil, err
	}

	fd, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	return fd, nil
}

func TouchFile(file string) error {
	path := filepath.Dir(file)
	if err := MkdirAll(path); err != nil {
		return err
	}

	fd, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	return fd.Close()
}

func WriteFile(file string, data []byte) error {
	path := filepath.Dir(file)
	if err := MkdirAll(path); err != nil {
		return err
	}

	return os.WriteFile(file, data, 0644)
}

func WriteFileByIO(file string, r io.Reader) error {
	path := filepath.Dir(file)
	if err := MkdirAll(path); err != nil {
		return err
	}

	fd, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	_, err = io.Copy(fd, r)

	return err
}

func MkdirAll(path string) error {
	if info, err := os.Stat(path); os.IsNotExist(err) || !info.IsDir() {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}

func GetFilesInDir(dir string, suffix ...string) ([]fs.FileInfo, error) {
	if err := MkdirAll(dir); err != nil {
		return nil, err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	f := make([]fs.FileInfo, 0, 2)
	filter := ""
	if len(suffix) > 0 {
		filter = suffix[0]
	}
	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			return nil, err
		}
		if fileInfo.IsDir() {
			continue
		}
		if filter == "" {
			f = append(f, fileInfo)
		} else if strings.HasSuffix(file.Name(), filter) {
			f = append(f, fileInfo)
		}
	}
	return f, nil
}

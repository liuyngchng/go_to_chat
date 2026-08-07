package kb

import (
	"io"
	"os"
)

// FileStore 文件存储接口。
// 单例模式：LocalFileStore（本地文件系统）。
// 集群模式：S3FileStore（MinIO / S3 / 阿里云 OSS）。
type FileStore interface {
	// Save 保存文件（由 io.Reader 写入），返回写入字节数
	Save(path string, reader io.Reader) (int64, error)

	// ReadAll 读取整个文件内容
	ReadAll(path string) ([]byte, error)

	// Delete 删除文件
	Delete(path string) error

	// Exists 检查文件是否存在
	Exists(path string) (bool, error)

	// MkdirAll 创建目录（递归），已存在不报错
	MkdirAll(path string) error

	// Open 打开文件用于读取（返回 io.ReadCloser）
	Open(path string) (io.ReadCloser, error)
}

// LocalFileStore 本地文件系统实现。
// 直接代理 os 包操作，额外处理 MkdirAll。
type LocalFileStore struct{}

// NewLocalFileStore 创建本地文件存储
func NewLocalFileStore() *LocalFileStore {
	return &LocalFileStore{}
}

// Save 保存文件到本地
func (s *LocalFileStore) Save(path string, reader io.Reader) (int64, error) {
	if err := os.MkdirAll(dirOf(path), 0755); err != nil {
		return 0, err
	}

	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return io.Copy(f, reader)
}

// ReadAll 读取本地文件
func (s *LocalFileStore) ReadAll(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Delete 删除本地文件
func (s *LocalFileStore) Delete(path string) error {
	return os.Remove(path)
}

// Exists 检查本地文件是否存在
func (s *LocalFileStore) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// MkdirAll 创建本地目录
func (s *LocalFileStore) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

// Open 打开本地文件
func (s *LocalFileStore) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// dirOf 返回路径的父目录
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

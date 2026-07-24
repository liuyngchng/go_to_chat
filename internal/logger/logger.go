package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	// Logger 全局 logger 实例
	Logger *slog.Logger
)

// Init 初始化日志，同时输出到控制台和文件
func Init(debug bool) error {
	// 确保日志目录存在
	logDir := filepath.Dir("app.log")
	if logDir != "." {
		os.MkdirAll(logDir, 0755)
	}

	f, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// 控制台 + 文件双输出
	multiWriter := io.MultiWriter(os.Stdout, f)

	// 日志级别
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	Logger = slog.New(slog.NewTextHandler(multiWriter, opts))

	// 设置为默认 logger
	slog.SetDefault(Logger)

	return nil
}

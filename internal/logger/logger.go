package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

var (
	// Logger 全局 logger 实例
	Logger *slog.Logger
)

// customHandler 自定义日志格式
// 格式: 2026-07-25 09:39:40.638 INFO [manager.go:431] 消息内容 key=value ...
type customHandler struct {
	w      io.Writer
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
}

func (h *customHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *customHandler) Handle(_ context.Context, r slog.Record) error {
	// 1. 时间
	buf := []byte(r.Time.Format("2006-01-02 15:04:05.000"))

	// 2. 级别
	buf = append(buf, ' ')
	buf = append(buf, r.Level.String()...)

	// 3. 源码位置
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		buf = append(buf, ' ')
		buf = append(buf, '[')
		buf = append(buf, filepath.Base(f.File)...)
		buf = append(buf, ':')
		buf = strconv.AppendInt(buf, int64(f.Line), 10)
		buf = append(buf, ']')
	}

	// 4. 消息
	buf = append(buf, ' ')
	buf = append(buf, r.Message...)

	// 5. 属性 (key=value)
	r.Attrs(func(a slog.Attr) bool {
		buf = append(buf, ' ')
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		buf = append(buf, a.Value.String()...)
		return true
	})

	buf = append(buf, '\n')

	_, err := h.w.Write(buf)
	return err
}

func (h *customHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &customHandler{
		w:     h.w,
		level: h.level,
		attrs: append(h.attrs, attrs...),
	}
}

func (h *customHandler) WithGroup(name string) slog.Handler {
	return &customHandler{
		w:      h.w,
		level:  h.level,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}

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

	// 使用自定义格式 handler
	handler := &customHandler{w: multiWriter, level: level}

	// 添加 source 到每条日志（通过 WithAttrs 预注入 PC 采集）
	Logger = slog.New(handler)

	// 设置为默认 logger
	slog.SetDefault(Logger)

	return nil
}

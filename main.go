package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go_to_chat/internal/config"
	"go_to_chat/internal/handler"
	"go_to_chat/internal/kb"
	"go_to_chat/internal/logger"
	"go_to_chat/internal/session"
	"go_to_chat/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.Load("cfg.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := logger.Init(cfg.Server.Debug); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	slog.Info("启动 AI 客服系统...")

	// 初始化 SQLite 元数据存储
	metaStore, err := store.NewSQLiteStore("cfg.db")
	if err != nil {
		slog.Error("初始化数据库失败", "error", err)
		os.Exit(1)
	}
	defer metaStore.Close()

	// 初始化会话管理器
	sessionMgr := session.NewManager()

	// 初始化知识库管理器
	kbManager := kb.NewManager(cfg, metaStore)

	// 启动后台文档处理 worker
	go kbManager.StartFileWorker()

	// 初始化 HTTP 处理器
	h := handler.New(cfg, kbManager, sessionMgr, metaStore)

	// 设置 Gin 路由
	if !cfg.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// 加载 HTML 模板
	r.LoadHTMLGlob("web/templates/*")

	// 静态文件
	r.Static("/static", "./web/static")

	// 页面路由
	r.GET("/", h.Page.Index)
	r.GET("/vdb/idx", h.Page.VdbIndex)

	// 聊天 API
	r.POST("/chat", h.Chat.Chat)
	r.POST("/chat/clear", h.Chat.Clear)

	// 知识库管理 API
	vdb := r.Group("/vdb")
	vdb.POST("/my/list", h.Vdb.MyList)
	vdb.POST("/pub/list", h.Vdb.PubList)
	vdb.POST("/file/list", h.Vdb.FileList)
	vdb.POST("/set/default", h.Vdb.SetDefault)
	vdb.POST("/create", h.Vdb.Create)
	vdb.POST("/delete", h.Vdb.Delete)
	vdb.POST("/upload", h.Vdb.Upload)
	vdb.POST("/process/info", h.Vdb.ProcessInfo)
	vdb.POST("/search", h.Vdb.Search)
	vdb.POST("/file/delete", h.Vdb.FileDelete)

	// 优雅退出
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("正在关闭服务...")
		kbManager.StopFileWorker()
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("服务启动", "addr", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("服务启动失败", "error", err)
		os.Exit(1)
	}
}

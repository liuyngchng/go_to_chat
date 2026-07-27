package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
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

//go:embed web
var webFS embed.FS

func main() {
	// 加载配置（server + milvus 从 YAML 文件）
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

	slog.Info("启动对话机器人...")

	// 检查 cfg.db 是否存在，必须由部署人员从 cfg.db.template 手动复制
	if _, err := os.Stat("cfg.db"); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "错误: cfg.db 不存在，请将 cfg.db.template 复制为 cfg.db 后重新启动\n")
		os.Exit(1)
	}

	// 初始化 SQLite 元数据存储
	metaStore, err := store.NewSQLiteStore("cfg.db")
	if err != nil {
		slog.Error("初始化数据库失败", "error", err)
		os.Exit(1)
	}
	defer metaStore.Close()

	// 从 SQLite 加载运行时配置（sys、api），YAML 值作为种子
	if err := config.LoadRuntimeConfig(metaStore, cfg); err != nil {
		slog.Error("加载运行时配置失败", "error", err)
		os.Exit(1)
	}

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
	r := gin.New()
	r.Use(logger.GinLogger(), gin.Recovery())

	// 加载 HTML 模板（从 embed.FS）
	tmpl, err := template.ParseFS(webFS, "web/templates/*")
	if err != nil {
		slog.Error("加载模板失败", "error", err)
		os.Exit(1)
	}
	r.SetHTMLTemplate(tmpl)

	// 静态文件（从 embed.FS）
	staticFS, err := fs.Sub(webFS, "web/static")
	if err != nil {
		slog.Error("加载静态文件失败", "error", err)
		os.Exit(1)
	}
	r.StaticFS("/static", http.FS(staticFS))

	// 页面路由
	r.GET("/", h.Page.Index)
	r.GET("/vdb/idx", h.Page.VdbIndex)
	r.GET("/admin/config", h.Page.ConfigIndex)

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

	// 系统配置 API
	r.GET("/api/config", h.Config.GetConfig)
	r.POST("/api/config", h.Config.UpdateConfig)

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

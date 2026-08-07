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

	"kb-chat-flow/internal/config"
	"kb-chat-flow/internal/handler"
	"kb-chat-flow/internal/kb"
	"kb-chat-flow/internal/logger"
	"kb-chat-flow/internal/session"
	"kb-chat-flow/internal/store"

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

	// 初始化元数据存储（根据 store.backend 配置选择后端）
	var metaStore store.MetaStore
	switch cfg.Store.Backend {
	case "mysql":
		if cfg.MySQL.DSN == "" {
			fmt.Fprintf(os.Stderr, "错误: store.backend=mysql 但 mysql.dsn 为空\n")
			os.Exit(1)
		}
		slog.Info("使用 MySQL 存储")
		ms, err := store.NewMySQLStore(cfg.MySQL.DSN)
		if err != nil {
			slog.Error("初始化 MySQL 失败", "error", err)
			os.Exit(1)
		}
		metaStore = ms
	default:
		// "sqlite" 或空
		if _, err := os.Stat("cfg.db"); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "错误: cfg.db 不存在，请将 cfg.db.template 复制为 cfg.db 后重新启动\n")
			os.Exit(1)
		}
		slog.Info("使用 SQLite 存储")
		ss, err := store.NewSQLiteStore("cfg.db")
		if err != nil {
			slog.Error("初始化 SQLite 失败", "error", err)
			os.Exit(1)
		}
		metaStore = ss
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

	// API 调用日志中间件（记录携带 Authorization 头的请求）
	r.Use(handler.ApiCallLogMiddleware(metaStore))

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

	// 免认证路由
	r.GET("/login", h.Auth.LoginPage)

	// 认证 API（JSON）
	r.POST("/api/login", h.Auth.Login)
	r.POST("/api/logout", h.Auth.Logout)

	// 需要认证的页面路由
	authPage := r.Group("/")
	authPage.Use(h.Auth.AuthMiddleware())
	{
		authPage.GET("/", h.Page.Index)
		authPage.GET("/vdb/idx", h.Page.VdbIndex)
		authPage.GET("/user/api", h.Page.UserApiIndex)
	}

	// 需要认证的 API 路由（受 sys.api_auth 开关控制）
	authAPI := r.Group("/api")
	authAPI.Use(h.Auth.ApiAuthMiddleware())
	{
		// 聊天
		authAPI.POST("/chat", h.Chat.Chat)
		authAPI.GET("/chat/history", h.Chat.History)
		authAPI.POST("/chat/clear", h.Chat.Clear)

		// 在线座席
		authAPI.GET("/agents", h.Auth.GetOnlineAgents)

		// FAQ（所有认证用户可读）
		authAPI.GET("/faq", h.Faq.List)
		authAPI.GET("/faq/template", h.Faq.Template)
		// 当前用户信息
		authAPI.GET("/me", h.Auth.Me)

		// AI Agent 公开列表（聊天页下拉选择用）
		authAPI.GET("/ai-agents/public", h.Agent.ListPublic)

		// 系统变量列表（供创建 Agent 时参考可用变量）
		authAPI.GET("/system-vars", h.Agent.ListSystemVars)

		// 工作流公开列表（聊天页下拉选择用）
		authAPI.GET("/workflows", h.Workflow.ListPublic)

		// 系统配置（读取）
		authAPI.GET("/config", h.Config.GetConfig)

		// 知识库（VDB）
		authAPI.GET("/vdb", h.Vdb.MyList)
		authAPI.GET("/vdb/pub", h.Vdb.PubList)
		authAPI.POST("/vdb", h.Vdb.Create)
		authAPI.DELETE("/vdb/:id", h.Vdb.Delete)
		authAPI.PUT("/vdb/:id/default", h.Vdb.SetDefault)
		authAPI.GET("/vdb/:id/files", h.Vdb.FileList)
		authAPI.POST("/vdb/:id/upload", h.Vdb.Upload)
		authAPI.POST("/vdb/search", h.Vdb.Search)
		authAPI.GET("/vdb/file/:id/progress", h.Vdb.ProcessInfo)
		authAPI.DELETE("/vdb/file/:id", h.Vdb.FileDelete)
	}

	// 管理员专属页面路由
	adminPage := r.Group("/admin")
	adminPage.Use(h.Auth.AuthMiddleware(), h.Auth.AdminOnlyMiddleware())
	{
		adminPage.GET("/config", h.Page.ConfigIndex)
	}

	// 管理员：业务知识库绑定页面（独立入口，嵌入系统管理页）
	adminPage.GET("/vdb/bind", h.Page.VdbBindIndex)

	// 管理员专属 API
	adminAPI := r.Group("/api")
	adminAPI.Use(h.Auth.ApiAuthMiddleware(), h.Auth.AdminOnlyMiddleware())
	{
		adminAPI.PUT("/config", h.Config.UpdateConfig)

		// 模型连接测试
		adminAPI.POST("/config/test-models", h.Config.TestModels)

		// csm 业务知识库绑定（仅管理员可改）
		adminAPI.GET("/vdb/bindings", h.Vdb.BindingGet)
		adminAPI.PUT("/vdb/bindings", h.Vdb.BindingPut)

		// FAQ 管理
		adminAPI.POST("/faq", h.Faq.Create)
		adminAPI.POST("/faq/upload", h.Faq.Upload)
		adminAPI.PUT("/faq/:id", h.Faq.Update)
		adminAPI.DELETE("/faq/:id", h.Faq.Delete)
		adminAPI.DELETE("/faq", h.Faq.ClearAll)

		// 用户管理
		adminAPI.GET("/users", h.User.ListUsers)
		adminAPI.POST("/users", h.User.CreateUser)
		adminAPI.DELETE("/users/:name", h.User.DeleteUser)
		adminAPI.PUT("/users/:name/reset-pwd", h.User.ResetUserPwd)

		// AI Agent 管理
		adminAPI.GET("/ai-agents", h.Agent.List)
		adminAPI.POST("/ai-agents", h.Agent.Create)
		adminAPI.GET("/ai-agents/:id", h.Agent.Get)
		adminAPI.PUT("/ai-agents/:id", h.Agent.Update)
		adminAPI.DELETE("/ai-agents/:id", h.Agent.Delete)

		// 工作流管理
		adminAPI.POST("/workflows", h.Workflow.Create)
		adminAPI.GET("/workflows/:id", h.Workflow.Get)
		adminAPI.PUT("/workflows/:id", h.Workflow.Update)
		adminAPI.DELETE("/workflows/:id", h.Workflow.Delete)

		// 意图分类测试
		adminAPI.POST("/classifier/test", h.Chat.TestClassifier)
	}

	// 用户自助 API（受 sys.api_auth 开关控制）
	userAPI := r.Group("/api/user")
	userAPI.Use(h.Auth.ApiAuthMiddleware())
	{
		userAPI.PUT("/password", h.User.ChangePassword)
		userAPI.GET("/tokens", h.User.ListMyTokens)
		userAPI.POST("/token", h.User.GenerateToken)
		userAPI.GET("/call-logs", h.User.MyCallLogs)
	}

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
	slog.Info("服务启动", "addr", addr, "url", fmt.Sprintf("http://localhost%s", addr))
	if err := r.Run(addr); err != nil {
		slog.Error("服务启动失败", "error", err)
		os.Exit(1)
	}
}
